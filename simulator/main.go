package main

import (
	"context"
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
)

func main() {
	brokerUrl := os.Getenv("BROKER_URL")
	if brokerUrl == "" {
		log.Fatalln("[SIMULATOR_MAIN] missing BROKER_URL env variable")
		os.Exit(1)
	}

	// Get DBC file path from environment variable, or use default
	dbcPath := os.Getenv("DBC_CONFIG_PATH")
	if dbcPath == "" {
		dbcPath = "config.dbc"
	}

	// Load topics from DBC file using vera package
	topics, err := loadTopicsFromDBC(dbcPath)
	if err != nil {
		log.Fatalf("[SIMULATOR_MAIN] failed to load topics from DBC: %s\n", err.Error())
		os.Exit(1)
	}
	log.Printf("[SIMULATOR_MAIN] using %d topics: %v\n", len(topics), topics)

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
