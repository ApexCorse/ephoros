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

	"github.com/ApexCorse/ephoros/server/internal/mqtt"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

func main() {
	brokerUrl := os.Getenv("BROKER_URL")
	if brokerUrl == "" {
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

	topics := []string{
		"Battery/Module-1/NTC-1",
		"Battery/Module-1/NTC-2",
		"Battery/Module-2/NTC-1",
		"Engine/Module-1/NTC-1",
		"Engine/Module-1/NTC-3",
		"Engine/Module-4/NTC-1",
		"Engine/Module-2/NTC-2",
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
		}

		time.Sleep(time.Duration(interval) * time.Millisecond)
	}
}

func generateRandomData() ([]byte, error) {
	value := rand.Float32()*1000 - 500

	jsonPayload := struct {
		Value float32   `json:"value"`
		Time  time.Time `json:"time"`
		Unit  string    `json:"unit"`
	}{
		Value: value,
		Time:  time.Now(),
		Unit:  "V",
	}

	data, err := json.Marshal(jsonPayload)
	if err != nil {
		return nil, err
	}

	return data, nil
}
