package stopalerts

import (
	"time"

	"github.com/google/uuid"
	"github.com/shukisha/backend/internal/routes"
)

type StopAlert struct {
	ID              uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	UserID          uuid.UUID  `gorm:"type:uuid;not null"`
	RouteID         uuid.UUID  `gorm:"type:uuid;not null"`
	StopID          uuid.UUID  `gorm:"type:uuid;not null"`
	Stop            routes.Stop `gorm:"foreignKey:StopID"`
	DriverID        *uuid.UUID `gorm:"type:uuid"` // Optional: link to specific driver
	Status          string     `gorm:"size:20;default:'ARMED'"` // ARMED, APPROACHING, FULFILLED, EXPIRED, CANCELLED
	TriggerRadius   int        `gorm:"default:500"` // distance in meters
	TriggeredAt     *time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CreateAlertRequest struct {
	RouteID  string `json:"route_id"`
	StopID   string `json:"stop_id"`
	DriverID string `json:"driver_id,omitempty"`
	Radius   int    `json:"radius,omitempty"`
}
