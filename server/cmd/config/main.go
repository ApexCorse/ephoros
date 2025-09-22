package main

import (
	"log"
	"os"

	"github.com/ApexCorse/ephoros/server/internal/config"
	"github.com/ApexCorse/ephoros/server/internal/db"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	dbUrl := os.Getenv("DB_URL")
	if dbUrl == "" {
		log.Fatalln("[CONFIG_MAIN] missing env variables")
		os.Exit(1)
	}

	gormDb, err := gorm.Open(postgres.Open(dbUrl))
	if err != nil {
		log.Fatalf("[CONFIG_MAIN] couldn't open db: %s\n", err.Error())
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

	log.Println("[CONFIG_MAIN] loading configuration from 'configuration.json'")
	configFile, err := os.Open("configuration.json")
	if err != nil {
		log.Fatalln("[CONFIG_MAIN] couldn't find configuration file")
		os.Exit(1)
	}

	log.Println("[CONFIG_MAIN] parsing configuration")
	cfg, err := config.NewConfigFromReader(configFile)
	if err != nil {
		log.Fatalf("[CONFIG_MAIN] invalid configuration: %s\n", err.Error())
		os.Exit(1)
	}
	log.Println("[CONFIG_MAIN] configuration parsed successfully")

	configManager := config.NewConfigManager(cfg, customDb)
	if err = configManager.UpdateDB(); err != nil {
		log.Fatalf("[CONFIG_MAIN] could not update db based on configuration: %s\n", err.Error())
		os.Exit(1)
	}

	log.Println("[CONFIG_MAIN] configuration successfull")
}
