package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/ApexCorse/ephoros/server/internal/api"
	"github.com/ApexCorse/ephoros/server/internal/db"
	"github.com/ApexCorse/ephoros/server/internal/utils"
	"github.com/gorilla/mux"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	healtcheck := utils.NewHealtcheck()
	http.HandleFunc("/readyz", healtcheck.ReadyzHandler)
	go http.ListenAndServe(":6969", nil)

	dbUrl := os.Getenv("DB_URL")
	apiAddress := os.Getenv("API_ADDRESS")
	if dbUrl == "" {
		log.Fatalln("[API_MAIN] missing env variables")
		os.Exit(1)
	}

	if apiAddress == "" {
		apiAddress = ":8080"
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	gormDb, err := gorm.Open(postgres.Open(dbUrl))
	if err != nil {
		log.Fatalf("[API_MAIN] couldn't open db: %s\n", err.Error())
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

	log.Printf("[API_MAIN] starting API on port %s\n", apiAddress)
	api := api.NewAPI(&api.APIConfig{
		DB:      customDb,
		Router:  mux.NewRouter(),
		Address: apiAddress,
	})
	go api.Start()
	log.Println("[API_MAIN] started server")

	healtcheck.SetReady(true)

	<-ctx.Done()
}
