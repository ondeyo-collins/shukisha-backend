package ledger

import (
	"time"

	"github.com/google/uuid"
)

type LedgerEntry struct {
	ID          uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DriverID    uuid.UUID `gorm:"type:uuid;not null;index" json:"driver_id"`
	ParcelID    uuid.UUID `gorm:"type:uuid" json:"parcel_id"`
	Amount      float64   `gorm:"type:decimal(12,2)" json:"amount"`
	EntryType   string    `gorm:"size:20;default:'CREDIT'" json:"entry_type"` // CREDIT, DEBIT
	Description string    `gorm:"type:text" json:"description"`
	Status      string    `gorm:"size:20;default:'UNPAID'" json:"status"` // UNPAID, PAID
	CreatedAt   time.Time `json:"created_at"`
}

type PayoutBatch struct {
	ID            uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	DriverID      uuid.UUID `gorm:"type:uuid;not null" json:"driver_id"`
	TotalAmount   float64   `gorm:"type:decimal(12,2)" json:"total_amount"`
	MPesaReceipt  string    `gorm:"size:100" json:"mpesa_receipt"`
	Status        string    `gorm:"size:20;default:'COMPLETED'" json:"status"`
	CreatedAt     time.Time `json:"created_at"`
}
