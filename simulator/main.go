package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// Database models
type Record struct {
	ID        uint      `gorm:"primarykey" json:"id"`
	CreatedAt time.Time `json:"created_at"`
	Value     float32   `json:"value"`
	Unit      string    `json:"unit"`
	SensorID  uint      `json:"sensor_id"`
}

type Sensor struct {
	ID        uint      `gorm:"primarykey" json:"name"`
	Name      string    `gorm:"uniqueIndex:sensor_name_module" json:"name"`
	CreatedAt time.Time `json:"created_at"`
	Topic     string
	Records   []Record
	ModuleID  uint `gorm:"uniqueIndex:sensor_name_module"`
}

type Module struct {
	ID        uint     `gorm:"primarykey" json:"id"`
	Name      string   `gorm:"uniqueIndex:module_name_section" json:"name"`
	Sensors   []Sensor
	SectionID uint `gorm:"uniqueIndex:module_name_section"`
}

type Section struct {
	ID      uint     `gorm:"primarykey" json:"id"`
	Name    string   `gorm:"uniqueIndex" json:"name"`
	Modules []Module
}

type User struct {
	Token     string    `gorm:"primarykey" json:"token"`
	CreatedAt time.Time `json:"created_at"`
	Username  string    `gorm:"index" json:"username"`
}

// MQTTClient wraps the autopaho connection manager
type MQTTClient struct {
	c *autopaho.ConnectionManager
}

// Publish sends a message to the specified topic
func (c *MQTTClient) Publish(ctx context.Context, topic string, payload []byte) error {
	if topic == "" {
		return fmt.Errorf("topic cannot be empty")
	}
	if len(payload) == 0 {
		return fmt.Errorf("payload cannot be empty")
	}

	p := &paho.Publish{
		Topic:   topic,
		Payload: payload,
	}

	if _, err := c.c.Publish(ctx, p); err != nil {
		return err
	}
	return nil
}

// newMQTTClient creates a new MQTT client from the autopaho connection manager
func newMQTTClient(c *autopaho.ConnectionManager) *MQTTClient {
	return &MQTTClient{c: c}
}

// MQTTClientBuilder helps build MQTT clients
type MQTTClientBuilder struct {
	cfg *autopaho.ClientConfig
}

// newMQTTClientBuilder creates a new builder
func newMQTTClientBuilder(cfg *autopaho.ClientConfig) *MQTTClientBuilder {
	builder := &MQTTClientBuilder{}
	if cfg != nil {
		builder.cfg = cfg
	} else {
		builder.cfg = &autopaho.ClientConfig{}
	}
	return builder
}

func (b *MQTTClientBuilder) addServers(urls []*url.URL) *MQTTClientBuilder {
	b.cfg.ServerUrls = urls
	return b
}

func (b *MQTTClientBuilder) addKeepAlive(value uint16) *MQTTClientBuilder {
	b.cfg.KeepAlive = value
	return b
}

func (b *MQTTClientBuilder) addCleanStartOnInitialConnection(value bool) *MQTTClientBuilder {
	b.cfg.CleanStartOnInitialConnection = value
	return b
}

func (b *MQTTClientBuilder) addOnConnectionUp(f func(cm *autopaho.ConnectionManager, connAck *paho.Connack)) *MQTTClientBuilder {
	b.cfg.OnConnectionUp = f
	return b
}

func (b *MQTTClientBuilder) addOnConnectionError(f func(err error)) *MQTTClientBuilder {
	b.cfg.OnConnectError = f
	return b
}

func (b *MQTTClientBuilder) addClientId(id string) *MQTTClientBuilder {
	b.cfg.ClientConfig.ClientID = id
	return b
}

func (b *MQTTClientBuilder) build(ctx context.Context) (*MQTTClient, error) {
	cm, err := autopaho.NewConnection(ctx, *b.cfg)
	if err != nil {
		return nil, err
	}

	err = cm.AwaitConnection(ctx)
	if err != nil {
		return nil, err
	}

	return newMQTTClient(cm), nil
}

// getAllTopics retrieves all sensor topics from the database
func getAllTopics(db *gorm.DB) ([]string, error) {
	sensors := make([]Sensor, 0)

	if err := db.Select("topic").Find(&sensors).Error; err != nil {
		return nil, fmt.Errorf("couldn't retrieve sensors: %s", err.Error())
	}

	topics := make([]string, len(sensors))
	for i := range sensors {
		topics[i] = sensors[i].Topic
	}

	return topics, nil
}

func main() {
	brokerUrl := os.Getenv("BROKER_URL")
	dbUrl := os.Getenv("DB_URL")
	if brokerUrl == "" || dbUrl == "" {
		log.Fatalln("[SIMULATOR_MAIN] missing env variables")
		os.Exit(1)
	}

	intervalStr := os.Getenv("SIMULATOR_INTERVAL")
	interval := 1000
	if intervalStr != "" {
		newInterval, err := strconv.Atoi(intervalStr)
		if err == nil {
			interval = newInterval
		}
	}

	parsedUrl, err := url.Parse(brokerUrl)
	if err != nil {
		log.Fatalf("[SIMULATOR_MAIN] couldn't parse url: %s\n", err.Error())
		os.Exit(1)
	}

	gormDb, err := gorm.Open(postgres.Open(dbUrl))
	if err != nil {
		log.Fatalf("[SIMULATOR_MAIN] couldn't open db: %s\n", err.Error())
		os.Exit(1)
	}
	gormDb.AutoMigrate(
		&User{},
		&Section{},
		&Module{},
		&Sensor{},
		&Record{},
	)

	topics, err := getAllTopics(gormDb)
	if err != nil {
		log.Fatalf("[SIMULATOR_MAIN] couldn't get topics: %s\n", err.Error())
		os.Exit(1)
	}
	log.Printf("[SIMULATOR_MAIN] got %d topics: %v\n", len(topics), topics)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("[SIMULATOR_MAIN] starting MQTT simulator")
	client, err := newMQTTClientBuilder(nil).
		addServers([]*url.URL{parsedUrl}).
		addKeepAlive(20).
		addCleanStartOnInitialConnection(false).
		addOnConnectionUp(func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			log.Println("[SIMULATOR_MAIN] MQTT connection up")
		}).
		addOnConnectionError(func(err error) {
			log.Printf("[SIMULATOR_MAIN] MQTT connection error: %s\n", err.Error())
		}).
		addClientId("simulator").
		build(ctx)
	if err != nil {
		log.Fatalf("[SIMULATOR_MAIN] couldn't create MQTT simulator: %s\n", err.Error())
		os.Exit(1)
	}
	log.Println("[SIMULATOR_MAIN] MQTT simulator started")

	for {
		select {
		case <-ctx.Done():
			return
		default:
			data, err := generateRandomData()
			if err != nil {
				log.Fatalf("[SIMULATOR_MAIN] couldn't generate data: %s\n", err.Error())
				os.Exit(1)
			}
			ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			i := rand.Intn(len(topics))
			topic := topics[i]
			topic = "data/" + topic

			if err := client.Publish(ctx, topic, data); err != nil {
				log.Fatalf("[SIMULATOR_MAIN] couldn't send data: %s\n", err.Error())
				os.Exit(1)
			}
			log.Printf("[SIMULATOR_MAIN] sent data to topic: %s\n", topic)

			<-ctx.Done()
		}

		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
}

func generateRandomData() ([]byte, error) {
	timestamp := time.Now()
	value := rand.Float32()*1000 - 500

	jsonPayload := struct {
		Value     float32   `json:"value"`
		Timestamp time.Time `json:"timestamp"`
		Unit      string    `json:"unit"`
	}{
		Value:     value,
		Timestamp: timestamp,
		Unit:      "V",
	}

	data, err := json.Marshal(jsonPayload)
	if err != nil {
		return nil, err
	}

	return data, nil
}
