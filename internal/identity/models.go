package identity

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID          uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PhoneNumber string         `gorm:"uniqueIndex;not null"`
	FullName    string         `gorm:"size:255"`
	IsActive    bool           `gorm:"default:true"`
	LastLogin   *time.Time
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type OTPChallenge struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	PhoneNumber string    `gorm:"not null"`
	CodeHash    string    `gorm:"not null"`
	Salt        string    `gorm:"not null"`
	ExpiresAt   time.Time `gorm:"not null"`
	VerifiedAt  *time.Time
	Attempts    int       `gorm:"default:0"`
	CreatedAt   time.Time
}
