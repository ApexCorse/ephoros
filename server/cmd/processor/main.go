package main

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/ApexCorse/ephoros/server/internal/mqtt"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

func main() {
	brokerUrl := os.Getenv("BROKER_URL")
	if brokerUrl == "" {
		log.Fatalln("[PROCESSOR_MAIN] missing env variables")
		os.Exit(1)
	}

	parsedUrl, err := url.Parse(brokerUrl)
	if err != nil {
		log.Fatalf("[PROCESSOR_MAIN] couldn't parse url: %s\n", err.Error())
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	log.Println("[PROCESSOR_MAIN] starting MQTT client")
	_, err = mqtt.NewMQTTClientBuilder(nil).
		AddServers([]*url.URL{parsedUrl}).
		AddKeepAlive(20).
		AddOnConnectionUp(func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			log.Println("[PROCESSOR_MAIN] MQTT connection up")

			if _, err := cm.Subscribe(ctx, &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{
						Topic: "raw/#",
					},
				},
			}); err != nil {
				log.Println("[PROCESSOR_MAIN] couldn't subscribe to topic: raw/#")
			}
		}).
		AddOnConnectionError(func(err error) {
			log.Printf("[PROCESSOR_MAIN] MQTT connection error: %s\n", err.Error())
		}).
		AddOnPublishReceived(func(pr paho.PublishReceived) (bool, error) {
			if !strings.HasPrefix(pr.Packet.Topic, "raw/") {
				return false, nil
			}
			log.Printf("[HandleAddRecordToDB] data incoming from topic: %s\n", pr.Packet.Topic)

			cleanTopic := strings.TrimPrefix(pr.Packet.Topic, "raw/")

			data := pr.Packet.Payload
			if len(data) != 12 {
				return false, fmt.Errorf("[HandleAddRecordToDB] invalid payload length: %v", data)
			}
			unsigned := binary.BigEndian.Uint32(data[8:])
			value := int32(unsigned)

			timestampUint64 := binary.BigEndian.Uint64(data[:8])
			timestamp := int64(timestampUint64)

			actualTime := time.UnixMilli(timestamp)

			jsonData := struct {
				Value     float32   `json:"value"`
				Timestamp time.Time `json:"timestamp"`
			}{
				Value:     float32(value),
				Timestamp: actualTime,
			}

			newPayload, err := json.Marshal(jsonData)
			if err != nil {
				return false, fmt.Errorf("[HandleAddRecordToDB] couldn't parse data: %s", err.Error())
			}

			ctx, stop := context.WithTimeout(ctx, 10*time.Second)
			defer stop()

			_, err = pr.Client.Publish(ctx, &paho.Publish{
				Topic:   fmt.Sprintf("clean/%s", cleanTopic),
				Payload: newPayload,
			})
			if err != nil {
				return false, fmt.Errorf("[HandleAddRecordToDB] couldn't publish clean data: %s", err.Error())
			}

			return true, nil
		}).
		AddClientId("processor").
		Build(ctx)
	if err != nil {
		log.Fatalf("[PROCESSOR_MAIN] couldn't create MQTT client: %s\n", err.Error())
		os.Exit(1)
	}
	log.Println("[PROCESSOR_MAIN] MQTT client started")

	<-ctx.Done()
}
