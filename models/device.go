package main

import (
	"github.com/google/uuid"
	"time"
)

type Device struct {
	ID       uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name     string    `gorm:"not null"`
	PubKey   string    `gorm:"not null;uniqueIndex"`
	Endpoint string    `gorm:"not null"`
	IP       string    `gorm:"type:inet;not null"`
	LastSeen time.Time `gorm:"not null"`
}
