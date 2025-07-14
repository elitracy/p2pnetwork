package models

import (
	"github.com/google/uuid"
	"time"
)

type Device struct {
	ID        uuid.UUID `gorm:"type:uuid;default:gen_random_uuid();primaryKey"`
	Name      string    `gorm:"not null"`
	PubKey    string    `gorm:"not null;uniqueIndex"`
	IP        string    `gorm:"type:inet;not null"`
	Port      string    `gorm:"not null"`
	Endpoint  string    `gorm:"not null"`
	LastSeen  time.Time `gorm:"not null"`
	Connected bool      `gorm:"not null"`
}
