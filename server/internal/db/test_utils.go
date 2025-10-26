package db

import (
	"fmt"
	"math/rand"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestDB() (*gorm.DB, func(), error) {
	baseDbUrl := os.Getenv("BASE_DB_URL")
	dbUrl := os.Getenv("DB_URL")

	dbName := fmt.Sprintf("test_db_%d", rand.Int())
	rootDB, err := gorm.Open(postgres.Open(dbUrl), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}

	if err := rootDB.Exec("CREATE DATABASE " + dbName).Error; err != nil {
		return nil, nil, err
	}

	dsnTest := baseDbUrl + dbName
	testDB, err := gorm.Open(postgres.Open(dsnTest), &gorm.Config{})
	if err != nil {
		rootDB.Exec("DROP DATABASE " + dbName)
		return nil, nil, err
	}

	// testDB.AutoMigrate(&Section{}, &Module{}, &Sensor{}, &Record{}, &User{})

	cleanup := func() {
		sqlDB, _ := testDB.DB()
		sqlDB.Close()
		rootDB.Exec("DROP DATABASE " + dbName)
		sqlRoot, _ := rootDB.DB()
		sqlRoot.Close()
	}

	return testDB, cleanup, nil
}
