package feedback

import (
	"time"

	"github.com/google/uuid"
)

type Rating struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	RaterID   uuid.UUID `gorm:"type:uuid;not null"`
	TargetID  uuid.UUID `gorm:"type:uuid;not null"`
	Score     int       `gorm:"type:int"`
	Comment   string    `gorm:"type:text"`
	CreatedAt time.Time
}

type Report struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	ReporterID  uuid.UUID `gorm:"type:uuid;not null"`
	ReferenceID string    `gorm:"size:100;not null"`
	Reason      string    `gorm:"size:255;not null"`
	Description string    `gorm:"type:text"`
	Status      string    `gorm:"size:20;default:'OPEN'"`
	CreatedAt   time.Time
}
