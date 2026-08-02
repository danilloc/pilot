package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Driver mirrors the drivers table. Fields tagged json:"-" are internal or
// security-sensitive and must never be serialized directly to API clients —
// call Redact() before returning a Driver in a response.
type Driver struct {
	ID     int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID   string `gorm:"type:varchar(36);uniqueIndex;not null" json:"uuid"`
	UberID string `gorm:"type:varchar(100);uniqueIndex;not null" json:"-"`

	Name              string `gorm:"type:varchar(255);not null" json:"name"`
	Email             string `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	Phone             string `gorm:"type:varchar(20)" json:"phone"`
	ProfilePictureURL string `gorm:"type:text" json:"profile_picture_url"`

	Rating        float64 `gorm:"type:decimal(3,2);default:0.00" json:"rating"`
	TotalTrips    int     `gorm:"default:0" json:"total_trips"`
	AccountStatus string  `gorm:"type:varchar(50);default:ACTIVE" json:"account_status"`

	OAuthToken        string     `gorm:"column:oauth_token;type:text" json:"-"`
	OAuthRefreshToken string     `gorm:"column:oauth_refresh_token;type:text" json:"-"`
	TokenExpiresAt    *time.Time `json:"-"`

	FailedLoginAttempts int        `gorm:"default:0" json:"-"`
	LastLoginIP         string     `gorm:"type:varchar(45)" json:"-"`
	LastLoginTime       *time.Time `json:"-"`
	IsBanned            bool       `gorm:"default:false" json:"-"`
	BanReason           string     `gorm:"type:varchar(255)" json:"-"`

	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
	LastSyncedAt *time.Time `json:"last_synced_at,omitempty"`
}

func (Driver) TableName() string { return "drivers" }

// BeforeCreate assigns a public UUID when one hasn't already been set.
func (d *Driver) BeforeCreate(tx *gorm.DB) error {
	if d.UUID == "" {
		d.UUID = uuid.NewString()
	}
	return nil
}

// Redact strips fields that must never leave the server before a Driver is
// serialized into an API response.
func (d *Driver) Redact() {
	d.OAuthToken = ""
	d.OAuthRefreshToken = ""
	d.FailedLoginAttempts = 0
	d.LastLoginIP = ""
	d.BanReason = ""
}

// Validate checks the business-level invariants required before a Driver
// can be persisted or returned to a client.
func (d *Driver) Validate() error {
	if !isValidEmail(d.Email) {
		return ErrInvalidEmail
	}
	if d.Phone != "" && !isValidE164(d.Phone) {
		return ErrInvalidPhone
	}
	if d.Rating < 0 || d.Rating > 5 {
		return ErrInvalidRating
	}
	if d.IsBanned {
		return ErrDriverBanned
	}
	return nil
}
