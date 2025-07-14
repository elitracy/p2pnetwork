package main

import (
	"errors"
	"log"
	"time"

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

func RegisterDevice(device models.Device) (*models.Device, error) {
	result := db.Create(&device)
	return &device, result.Error
}

func UpdateDeviceByPubKey(pubKey, endpoint, ip string) (*models.Device, error) {
	var device models.Device

	// Find device by PubKey
	result := db.First(&device)

	if errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return nil, nil
	}

	if result.Error != nil {
		return nil, result.Error
	}

	// Update the fields
	device.IP = ip
	device.Endpoint = endpoint
	device.LastSeen = time.Now().UTC()

	// Save the updated device
	if err := db.Save(&device).Error; err != nil {
		return nil, err
	}

	return &device, nil
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
