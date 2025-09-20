package main

import (
	"context"
	"log"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/ApexCorse/ephoros/server/internal/api"
	"github.com/ApexCorse/ephoros/server/internal/config"
	"github.com/ApexCorse/ephoros/server/internal/db"
	"github.com/ApexCorse/ephoros/server/internal/mqtt"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"github.com/gorilla/mux"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	brokerUrl := os.Getenv("BROKER_URL")
	dbUrl := os.Getenv("DB_URL")
	apiAddress := os.Getenv("API_ADDRESS")
	if brokerUrl == "" || dbUrl == "" {
		log.Fatalln("[MAIN] missing env variables")
		os.Exit(1)
	}

	if apiAddress == "" {
		apiAddress = ":8080"
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
	gormDb.AutoMigrate(
		&db.User{},
		&db.Section{},
		&db.Module{},
		&db.Sensor{},
		&db.Record{},
	)

	customDb := db.NewDB(gormDb)

	log.Println("[MAIN] loading configuration from 'configuration.json'")
	configFile, err := os.Open("configuration.json")
	if err != nil {
		log.Fatalln("[MAIN] couldn't find configuration file")
		os.Exit(1)
	}

	log.Println("[MAIN] parsing configuration")
	cfg, err := config.NewConfigFromReader(configFile)
	if err != nil {
		log.Fatalf("[MAIN] invalid configuration: %s\n", err.Error())
		os.Exit(1)
	}
	log.Println("[MAIN] configuration parsed successfully")

	mqttHandler := mqtt.NewMQTTHandler(customDb, cfg)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	mqttCtx, stop := context.WithTimeout(ctx, time.Second*10)
	defer stop()
	log.Println("[MAIN] starting MQTT client")
	_, err = mqtt.NewMQTTClientBuilder(nil).
		AddServers([]*url.URL{parsedUrl}).
		AddKeepAlive(20).
		AddOnConnectionUp(func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			log.Println("[MAIN] MQTT connection up")

			if _, err := cm.Subscribe(ctx, &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{
						Topic: "raw/#",
					},
				},
			}); err != nil {
				log.Println("[MAIN] couldn't subscribe to topic: raw/#")
			}
		}).
		AddOnConnectionError(func(err error) {
			log.Printf("[MAIN] MQTT connection error: %s\n", err.Error())
		}).
		AddClientId("ephoros").
		AddOnPublishReceived(mqttHandler.HandleAddRecordToDB).
		Build(mqttCtx)
	if err != nil {
		log.Fatalf("[MAIN] couldn't create MQTT client: %s\n", err.Error())
		os.Exit(1)
	}
	log.Println("[MAIN] MQTT client started")

	log.Printf("[MAIN] starting API on port %s\n", apiAddress)
	api := api.NewAPI(&api.APIConfig{
		Config: cfg,
		DB: customDb,
		Router: mux.NewRouter(),
		Address: apiAddress,
	})
	go api.Start()
	log.Println("[MAIN] started server")

	<-ctx.Done()
}
