package parcels

import (
	"time"

	"github.com/google/uuid"
)

// NotificationJob is an outbound notification queued for async delivery by the worker.
// MVP: only SMS is used.
type NotificationJob struct {
	ID uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`

	Channel string `gorm:"size:20;not null;default:'SMS'" json:"channel"` // SMS
	To      string `gorm:"size:50;not null;index" json:"to"`
	Message string `gorm:"type:text;not null" json:"message"`

	Status        string     `gorm:"size:20;not null;default:'QUEUED';index" json:"status"` // QUEUED, SENT, FAILED
	Attempts      int        `gorm:"not null;default:0" json:"attempts"`
	MaxAttempts   int        `gorm:"not null;default:5" json:"max_attempts"`
	NextAttemptAt time.Time  `gorm:"not null;index" json:"next_attempt_at"`
	LastError     string     `gorm:"type:text" json:"last_error"`
	SentAt        *time.Time `json:"sent_at"`

	ParcelID         *uuid.UUID `gorm:"type:uuid;index" json:"parcel_id,omitempty"`
	CreatedByAdminID *uuid.UUID `gorm:"type:uuid;index" json:"created_by_admin_id,omitempty"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
