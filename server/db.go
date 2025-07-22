package main

import (
	"errors"
	"log"

	"github.com/elitracy/p2pnetwork/shared"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var db *gorm.DB

func initDB() error {
	var err error
	dsn := "host=localhost user=serveradmin password=server-admin-password dbname=p2pnetwork port=5432 sslmode=disable"
	db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if db == nil || err != nil {
		log.Fatal("failed to connect to database:", err)
	}

	// Enable pgcrypto extension for UUID generation
	db.Exec(`CREATE EXTENSION IF NOT EXISTS "pgcrypto"`)

	// Migrate the schema
	err = db.AutoMigrate(&models.Device{})
	if err != nil {
		log.Fatal("failed to migrate:", err)
	}

	return err
}

func UpsertDevice(device models.Device) error {
	result := db.Save(&device).Error
	return result
}

func GetDeviceByPubKey(pubKey string) (*models.Device, error) {
	var device models.Device
	result := db.First(&device, "pub_key = ?", pubKey)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return &device, result.Error
}

func GetDeviceByIP(ip string) (*models.Device, error) {
	var device models.Device
	result := db.First(&device, "ip = ?", ip)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if result.Error != nil {
		return nil, result.Error
	}

	return &device, result.Error
}

func GetAllDevices() ([]models.Device, error) {
	var devices []models.Device

	result := db.Find(&devices)
	if result.Error != nil {
		return nil, result.Error
	}

	return devices, nil
}
