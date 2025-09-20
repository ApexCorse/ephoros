package main

import (
	"context"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/ApexCorse/ephoros/server/internal/db"
	"github.com/ApexCorse/ephoros/server/internal/mqtt"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	brokerUrl := os.Getenv("BROKER_URL")
	dbUrl := os.Getenv("DB_URL")
	if brokerUrl == "" || dbUrl == "" {
		log.Fatalln("[MAIN] missing env variables")
		os.Exit(1)
	}

	parsedUrl, err := url.Parse(brokerUrl)
	if err != nil {
		log.Fatalf("[MAIN] couldn't parse url: %s\n", err.Error())
		os.Exit(1)
	}

	gormDb, err := gorm.Open(postgres.Open(dbUrl))
	if err != nil {
		log.Fatalf("[MAIN] couldn't open db: %s\n", err.Error())
		os.Exit(1)
	}

	customDb := db.NewDB(gormDb)

	//TODO: Add config
	mqttHandler := mqtt.NewMQTTHandler(customDb, nil)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	_, err = mqtt.NewMQTTClientBuilder(nil).
		AddServers([]*url.URL{parsedUrl}).
		AddOnPublishReceived(mqttHandler.HandleAddRecordToDB).
		Build(ctx)
	if err != nil {
		log.Fatalf("[MAIN] couldn't create MQTT client: %s\n", err.Error())
		os.Exit(1)
	}

	<-ctx.Done()
}
