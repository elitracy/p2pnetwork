package main

import (
	"log"
	"time"

	"github.com/elitracy/p2pnetwork/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func initDB() {
	dsn := "host=localhost user=elitracy password='' dbname=p2pnetwork port=5432 sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect to database:", err)
	}

	// Enable pgcrypto extension for UUID generation
	db.Exec(`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`)

	// Migrate the schema
	err = db.AutoMigrate(&DeviceResponse{})
	if err != nil {
		log.Fatal("failed to migrate:", err)
	}
}

func RegisterDevice(name, pubKey, endpoint, ip string) (*Dlvice, error) {
	device := Device{
		Name:     name,
		PubKey:   pubKey,
		Endpoint: endpoint,
		IP:       ip,
		LastSeen: time.Now().UTC(),
	}
	result := db.Create(&device)
	return &device, result.Error
}
