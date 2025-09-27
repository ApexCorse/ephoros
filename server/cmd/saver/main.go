package main

import (
	"context"
	"log"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"syscall"

	"github.com/ApexCorse/ephoros/server/internal/db"
	"github.com/ApexCorse/ephoros/server/internal/mqtt"
	"github.com/ApexCorse/ephoros/server/internal/utils"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	healthcheck := utils.NewHealtcheck()
	http.HandleFunc("/readyz", healthcheck.ReadyzHandler)

	go http.ListenAndServe(":6969", nil)

	brokerUrl := os.Getenv("BROKER_URL")
	dbUrl := os.Getenv("DB_URL")
	if brokerUrl == "" || dbUrl == "" {
		log.Fatalln("[SAVER_MAIN] missing env variables")
		os.Exit(1)
	}

	parsedUrl, err := url.Parse(brokerUrl)
	if err != nil {
		log.Fatalf("[SAVER_MAIN] couldn't parse url: %s\n", err.Error())
		os.Exit(1)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	gormDb, err := gorm.Open(postgres.Open(dbUrl))
	if err != nil {
		log.Fatalf("[SAVER_MAIN] couldn't open db: %s\n", err.Error())
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

	log.Println("[SAVER_MAIN] starting saver")
	_, err = mqtt.NewMQTTClientBuilder(nil).
		AddServers([]*url.URL{parsedUrl}).
		AddKeepAlive(20).
		AddOnConnectionUp(func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			log.Println("[SAVER_MAIN] MQTT connection up")

			if _, err := cm.Subscribe(ctx, &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{
						Topic: "p/#",
					},
				},
			}); err != nil {
				log.Println("[SAVER_MAIN] couldn't subscribe to topic: #")
			}
		}).
		AddOnConnectionError(func(err error) {
			log.Printf("[SAVER_MAIN] MQTT connection error: %s\n", err.Error())
		}).
		AddClientId("saver").
		AddOnPublishReceived(func(pr paho.PublishReceived) (bool, error) {
			return mqtt.HandleAddRecordToDB(customDb, pr)
		}).
		Build(ctx)
	if err != nil {
		log.Fatalf("[SAVER_MAIN] couldn't create MQTT client: %s\n", err.Error())
		os.Exit(1)
	}
	log.Println("[SAVER_MAIN] saver started")

	healthcheck.SetReady(true)

	<-ctx.Done()
}
