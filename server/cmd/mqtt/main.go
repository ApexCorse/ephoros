package main

import (
	"context"
	"log"
	"net/url"
	"os"
	"os/signal"
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
		log.Fatalln("[MQTT_MAIN] missing env variables")
		os.Exit(1)
	}

	parsedUrl, err := url.Parse(brokerUrl)
	if err != nil {
		log.Fatalf("[MQTT_MAIN] couldn't parse url: %s\n", err.Error())
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	gormDb, err := gorm.Open(postgres.Open(dbUrl))
	if err != nil {
		log.Fatalf("[MQTT_MAIN] couldn't open db: %s\n", err.Error())
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

	mqttHandler := mqtt.NewMQTTHandler(customDb)

	mqttCtx, stop := context.WithTimeout(ctx, time.Second*10)
	defer stop()
	log.Println("[MQTT_MAIN] starting MQTT client")
	_, err = mqtt.NewMQTTClientBuilder(nil).
		AddServers([]*url.URL{parsedUrl}).
		AddKeepAlive(20).
		AddOnConnectionUp(func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			log.Println("[MQTT_MAIN] MQTT connection up")

			if _, err := cm.Subscribe(ctx, &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{
						Topic: "raw/#",
					},
				},
			}); err != nil {
				log.Println("[MQTT_MAIN] couldn't subscribe to topic: raw/#")
			}
		}).
		AddOnConnectionError(func(err error) {
			log.Printf("[MQTT_MAIN] MQTT connection error: %s\n", err.Error())
		}).
		AddClientId("ephoros").
		AddOnPublishReceived(mqttHandler.HandleAddRecordToDB).
		Build(mqttCtx)
	if err != nil {
		log.Fatalf("[MQTT_MAIN] couldn't create MQTT client: %s\n", err.Error())
		os.Exit(1)
	}
	log.Println("[MQTT_MAIN] MQTT client started")

	<-ctx.Done()
}
