package drivers

import (
	"time"

	"github.com/google/uuid"
)

type DriverProfile struct {
	ID                 uuid.UUID `gorm:"type:uuid;primaryKey"`
	VehiclePlate       string    `gorm:"size:20"`
	SaccoName          string    `gorm:"size:100"`
	VerificationStatus string    `gorm:"size:20;default:'PENDING'"`
	UpdatedAt          time.Time
}

type DriverSession struct {
	ID             uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	DriverID       uuid.UUID `gorm:"type:uuid;not null"`
	DriverProfile  *DriverProfile `gorm:"foreignKey:DriverID"`
	RouteID        uuid.UUID `gorm:"type:uuid;not null"`
	Status         string    `gorm:"size:20;default:'ONLINE'"`
	LastLat        float64   `gorm:"type:decimal(9,6)"`
	LastLng        float64   `gorm:"type:decimal(9,6)"`
	LastHeartbeat  time.Time `gorm:"default:CURRENT_TIMESTAMP"`
	CreatedAt      time.Time
}

type HeartbeatRequest struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}
