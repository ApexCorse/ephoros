package main

import (
	"context"
	"encoding/json"
	"log"
	"math/rand"
	"net/url"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"

	"github.com/ApexCorse/ephoros/server/internal/db"
	"github.com/ApexCorse/ephoros/server/internal/mqtt"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

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
		&db.User{},
		&db.Section{},
		&db.Module{},
		&db.Sensor{},
		&db.Record{},
	)

	customDb := db.NewDB(gormDb)

	topics, err := customDb.GetAllTopics()
	if err != nil {
		log.Fatalf("[SIMULATOR_MAIN] couldn't get topics: %s\n", err.Error())
		os.Exit(1)
	}
	log.Printf("[SIMULATOR_MAIN] got %d topics: %v\n", len(topics), topics)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("[SIMULATOR_MAIN] starting MQTT simulator")
	client, err := mqtt.NewMQTTClientBuilder(nil).
		AddServers([]*url.URL{parsedUrl}).
		AddKeepAlive(20).
		AddCleanStartOnInitialConnection(false).
		AddOnConnectionUp(func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			log.Println("[SIMULATOR_MAIN] MQTT connection up")
		}).
		AddOnConnectionError(func(err error) {
			log.Printf("[SIMULATOR_MAIN] MQTT connection error: %s\n", err.Error())
		}).
		AddClientId("simulator").
		Build(ctx)
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
