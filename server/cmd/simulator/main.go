package main

import (
	"context"
	cryptoRand "crypto/rand"
	"encoding/binary"
	"fmt"
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
			completeTopic := fmt.Sprintf("raw/%s", topic)

			if err := client.Publish(ctx, completeTopic, data); err != nil {
				log.Fatalf("[SIMULATOR_MAIN] couldn't send data: %s\n", err.Error())
				os.Exit(1)
			}
			log.Printf("[SIMULATOR_MAIN] sent data to topic: %s\n", completeTopic)

			<-ctx.Done()
		}

		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
}

func generateRandomData() ([]byte, error) {
	randomBytes := make([]byte, 4)
	if _, err := cryptoRand.Read(randomBytes); err != nil {
		return nil, err
	}

	timestamp := time.Now().UnixNano()
	timestampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBytes, uint64(timestamp))

	data := make([]byte, 12)
	copy(data[:8], timestampBytes)
	copy(data[8:], randomBytes)

	return data, nil
}
