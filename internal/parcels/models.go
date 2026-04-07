package parcels

import (
	"time"

	"github.com/google/uuid"
)

type Parcel struct {
	ID             uuid.UUID             `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SenderID       uuid.UUID             `gorm:"type:uuid;not null" json:"sender_id"`
	DriverID       *uuid.UUID            `gorm:"type:uuid" json:"driver_id"`
	ReceiverPhone  string                `gorm:"size:20;not null" json:"receiver_phone"`
	Description    string                `gorm:"type:text" json:"description"`
	DeclaredValue  float64               `gorm:"type:decimal(12,2)" json:"declared_value"`
	Status         string                `gorm:"size:20;default:'DRAFT'" json:"status"`
	PickupOTP      string                `gorm:"size:6" json:"pickup_otp"`
	DeliveryOTP    string                `gorm:"size:6" json:"delivery_otp"`
	RouteID        uuid.UUID             `gorm:"type:uuid;not null" json:"route_id"`
	TrackingEvents []ParcelTrackingEvent `gorm:"foreignKey:ParcelID" json:"tracking_events"`
	Payment        *Payment              `gorm:"foreignKey:ParcelID" json:"payment"`
	CreatedAt      time.Time             `json:"created_at"`
	UpdatedAt      time.Time             `json:"updated_at"`
}

type ParcelTrackingEvent struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ParcelID  uuid.UUID `gorm:"type:uuid;not null" json:"parcel_id"`
	EventType string    `gorm:"size:50;not null" json:"event_type"` // BOOKED, PAID, PICKED_UP, IN_TRANSIT, DELIVERED
	Location  string    `gorm:"type:geography(POINT, 4326)" json:"location"`
	Remarks   string    `gorm:"type:text" json:"remarks"`
	CreatedAt time.Time `json:"created_at"`
}

type Payment struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ParcelID    uuid.UUID `gorm:"type:uuid;not null" json:"parcel_id"`
	Amount      float64   `gorm:"type:decimal(12,2)" json:"amount"`
	ProviderRef string    `gorm:"size:100" json:"provider_ref"` // e.g. M-Pesa Code
	Status      string    `gorm:"size:20;default:'PENDING'" json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

// ParcelNote is an internal-only note attached to a parcel by an admin operator.
type ParcelNote struct {
	ID        uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ParcelID  uuid.UUID `gorm:"type:uuid;not null;index" json:"parcel_id"`
	AdminID   uuid.UUID `gorm:"type:uuid;not null;index" json:"admin_id"`
	Note      string    `gorm:"type:text;not null" json:"note"`
	CreatedAt time.Time `json:"created_at"`
}

type ParcelClaimType string
type ParcelClaimStatus string

const (
	ParcelClaimTypeLost           ParcelClaimType = "LOST"
	ParcelClaimTypeDamaged        ParcelClaimType = "DAMAGED"
	ParcelClaimTypeDelayed        ParcelClaimType = "DELAYED"
	ParcelClaimTypeWrongRecipient ParcelClaimType = "WRONG_RECIPIENT"
)

const (
	ParcelClaimStatusOpen     ParcelClaimStatus = "OPEN"
	ParcelClaimStatusInReview ParcelClaimStatus = "IN_REVIEW"
	ParcelClaimStatusResolved ParcelClaimStatus = "RESOLVED"
	ParcelClaimStatusRejected ParcelClaimStatus = "REJECTED"
)

// ParcelClaim captures a dispute/claim raised against a parcel (admin-only for MVP).
type ParcelClaim struct {
	ID               uuid.UUID         `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ParcelID         uuid.UUID         `gorm:"type:uuid;not null;index" json:"parcel_id"`
	CreatedByAdminID uuid.UUID         `gorm:"type:uuid;not null;index" json:"created_by_admin_id"`
	Type             ParcelClaimType   `gorm:"size:30;not null" json:"type"`
	Status           ParcelClaimStatus `gorm:"size:30;not null;default:'OPEN'" json:"status"`
	Description      string            `gorm:"type:text;not null" json:"description"`
	ResolutionNotes  string            `gorm:"type:text" json:"resolution_notes"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

// ParcelClaimAttachment stores evidence for a claim. MVP uses Cloudinary (authenticated delivery).
type ParcelClaimAttachment struct {
	ID                uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	ClaimID           uuid.UUID `gorm:"type:uuid;not null;index" json:"claim_id"`
	UploadedByAdminID uuid.UUID `gorm:"type:uuid;not null;index" json:"uploaded_by_admin_id"`
	FileName          string    `gorm:"size:255;not null" json:"file_name"`
	MimeType          string    `gorm:"size:100;not null" json:"mime_type"`
	SizeBytes         int64     `gorm:"not null" json:"size_bytes"`

	CloudinaryPublicID     string `gorm:"size:255;not null;index" json:"cloudinary_public_id"`
	CloudinaryResourceType string `gorm:"size:50;not null" json:"cloudinary_resource_type"`
	CloudinaryType         string `gorm:"size:50;not null" json:"cloudinary_type"`
	CloudinaryVersion      int    `gorm:"not null" json:"cloudinary_version"`
	CloudinaryFormat       string `gorm:"size:50" json:"cloudinary_format"`

	CreatedAt time.Time `json:"created_at"`
}

type BookParcelRequest struct {
	RouteID       string  `json:"route_id"`
	ReceiverPhone string  `json:"receiver_phone"`
	Description   string  `json:"description"`
	DeclaredValue float64 `json:"declared_value"`
}
