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

	"github.com/ApexCorse/vera"
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

	config, err := getDbcConfig()
	if err != nil {
		log.Fatalf("[SIMULATOR_MAIN] couldn't load DBC config: %s\n", err.Error())
		os.Exit(1)
	}

	topics := getTopicsFromConfig(config)
	log.Printf("[SIMULATOR_MAIN] got %d topics: %v\n", len(topics), topics)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("[SIMULATOR_MAIN] starting MQTT simulator")
	client, err := NewMQTTClientBuilder(nil).
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

// getDbcConfig reads and parses the config.dbc file
func getDbcConfig() (*vera.Config, error) {
	dbcFilePath := os.Getenv("DBC_FILE_PATH")
	if dbcFilePath == "" {
		return nil, fmt.Errorf("DBC_FILE_PATH env var is not set")
	}

	dbcFile, err := os.Open(dbcFilePath)
	if err != nil {
		return nil, fmt.Errorf("error while opening DBC file: %w", err)
	}
	defer dbcFile.Close()

	config, err := vera.Parse(dbcFile)
	if err != nil {
		return nil, fmt.Errorf("error in parsing DBC file: %w", err)
	}

	return config, nil
}

// getTopicsFromConfig extracts MQTT topics from the vera config
func getTopicsFromConfig(config *vera.Config) []string {
	topics := make([]string, len(config.Topics))
	for i := range config.Topics {
		topics[i] = config.Topics[i].Topic
	}

	return topics
}
