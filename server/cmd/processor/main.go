package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/ApexCorse/ephoros/server/internal/mqtt"
	"github.com/ApexCorse/ephoros/server/internal/utils"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

func main() {
	healthcheck := utils.NewHealtcheck()
	http.HandleFunc("/readyz", healthcheck.ReadyzHandler)
	go http.ListenAndServe(":6969", nil)

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
			return mqtt.HandleProcessRawData(ctx, pr)
		}).
		AddClientId("processor").
		Build(ctx)
	if err != nil {
		log.Fatalf("[PROCESSOR_MAIN] couldn't create MQTT client: %s\n", err.Error())
		os.Exit(1)
	}
	log.Println("[PROCESSOR_MAIN] MQTT client started")

	healthcheck.SetReady(true)

	<-ctx.Done()
}
