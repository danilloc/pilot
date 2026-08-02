package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Trip mirrors the trips table. Coordinates are tagged json:"-": the API
// never returns raw lat/lng to clients (location privacy).
type Trip struct {
	ID         int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID       string `gorm:"type:varchar(36);uniqueIndex;not null" json:"uuid"`
	DriverID   int64  `gorm:"not null;index" json:"driver_id"`
	UberTripID string `gorm:"type:varchar(100);uniqueIndex;not null" json:"-"`

	StartedAt time.Time `gorm:"not null" json:"started_at"`
	EndedAt   time.Time `gorm:"not null;index" json:"ended_at"`

	DistanceKM float64 `gorm:"type:decimal(10,2);not null" json:"distance_km"`
	FareValue  float64 `gorm:"type:decimal(10,2);not null" json:"fare_value"`
	Currency   string  `gorm:"type:varchar(3);default:BRL" json:"currency"`

	FareBase     float64 `gorm:"type:decimal(10,2);default:0" json:"fare_base"`
	FareDistance float64 `gorm:"type:decimal(10,2);default:0" json:"fare_distance"`
	FareTime     float64 `gorm:"type:decimal(10,2);default:0" json:"fare_time"`
	TollCharge   float64 `gorm:"type:decimal(10,2);default:0" json:"toll_charge"`
	ServiceFee   float64 `gorm:"type:decimal(10,2);default:0" json:"service_fee"`

	City string `gorm:"type:varchar(100)" json:"city"`

	StartLat float64 `gorm:"column:start_latitude;type:decimal(10,8)" json:"-"`
	StartLng float64 `gorm:"column:start_longitude;type:decimal(11,8)" json:"-"`
	EndLat   float64 `gorm:"column:end_latitude;type:decimal(10,8)" json:"-"`
	EndLng   float64 `gorm:"column:end_longitude;type:decimal(11,8)" json:"-"`

	Status string `gorm:"type:varchar(50);default:COMPLETED;index" json:"status"`

	SyncedAt  time.Time `gorm:"autoCreateTime" json:"synced_at"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func (Trip) TableName() string { return "trips" }

// BeforeCreate assigns a public UUID when one hasn't already been set.
func (t *Trip) BeforeCreate(tx *gorm.DB) error {
	if t.UUID == "" {
		t.UUID = uuid.NewString()
	}
	return nil
}

// Validate checks the business-level invariants required before a Trip can
// be persisted.
func (t *Trip) Validate() error {
	if t.DistanceKM <= 0 {
		return ErrInvalidDistance
	}
	if t.FareValue < 0 {
		return ErrInvalidFare
	}
	if t.EndedAt.Before(t.StartedAt) {
		return ErrInvalidTimeRange
	}
	return nil
}
