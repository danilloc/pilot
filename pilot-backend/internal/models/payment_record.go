package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// PaymentRecord mirrors the payment_records table. net_amount is a
// MySQL-generated column (amount - tolls - taxes) and is intentionally not
// mapped here: the database, not the application, owns that value.
type PaymentRecord struct {
	ID       int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID     string `gorm:"type:varchar(36);uniqueIndex;not null" json:"uuid"`
	DriverID int64  `gorm:"not null;index" json:"driver_id"`

	UberPaymentID string  `gorm:"type:varchar(100);uniqueIndex" json:"-"`
	Amount        float64 `gorm:"type:decimal(10,2);not null" json:"amount"`
	Currency      string  `gorm:"type:varchar(3);default:BRL" json:"currency"`

	Tolls float64 `gorm:"type:decimal(10,2);default:0" json:"tolls"`
	Taxes float64 `gorm:"type:decimal(10,2);default:0" json:"taxes"`

	PaymentMethod string    `gorm:"type:varchar(50)" json:"payment_method"`
	PaymentDate   time.Time `gorm:"type:date;not null;index" json:"payment_date"`

	SyncedAt  time.Time `gorm:"autoCreateTime" json:"synced_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (PaymentRecord) TableName() string { return "payment_records" }

// BeforeCreate assigns a public UUID when one hasn't already been set.
func (p *PaymentRecord) BeforeCreate(tx *gorm.DB) error {
	if p.UUID == "" {
		p.UUID = uuid.NewString()
	}
	return nil
}
