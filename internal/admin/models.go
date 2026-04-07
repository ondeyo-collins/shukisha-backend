package admin

import (
	"time"

	"github.com/google/uuid"
)

type DashboardStats struct {
	TotalUsers     int64   `json:"total_users"`
	ActiveDrivers  int64   `json:"active_drivers"`
	ActiveParcels  int64   `json:"active_parcels"`
	ActiveAlerts   int64   `json:"active_alerts"`
	TotalRoutes    int64   `json:"total_routes"`
	TotalRevenue   float64 `json:"total_revenue"`
}

type AdminUserDetail struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	Email      string    `json:"email"`
	Name       string    `json:"name"`
	Role       string    `json:"role"`
	Department string    `json:"department"`
	Status     string    `json:"status"`
	CreatedAt  int64     `json:"created_at"`
}

type ManagementRequest struct {
	EntityID string `json:"entity_id"`
	Action   string `json:"action"` // APPROVE, SUSPEND, CANCEL, ACTIVATE
}

type AdminUser struct {
	ID         uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	UserID     uuid.UUID `gorm:"type:uuid;unique;not null;index" json:"user_id"`
	Role       string    `gorm:"size:20;default:'LOGISTICS'" json:"role"`
	Department string    `gorm:"size:50" json:"department"`
	Status     string    `gorm:"size:20;default:'ACTIVE'" json:"status"` // ACTIVE, SUSPENDED
	CreatedAt  int64     `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt  int64     `gorm:"autoUpdateTime" json:"updated_at"`
}

// AdminCredential stores email+password for admin web login
type AdminCredential struct {
	ID           uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AdminUserID  uuid.UUID `gorm:"type:uuid;unique;not null;index"`
	Email        string    `gorm:"size:255;unique;not null"`
	Name         string    `gorm:"size:255"`
	PasswordHash string    `gorm:"size:255;not null"`
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type AuditEvent struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
	AdminID   uuid.UUID `gorm:"type:uuid;not null;index"`
	Action    string    `gorm:"size:100;not null"`
	EntityID  string    `gorm:"size:100"`
	Details   string    `gorm:"type:text"`
	CreatedAt time.Time
}
