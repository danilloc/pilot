package repository

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"pilot-backend/internal/models"
)

// DriverRepository is the data-access layer for drivers.
type DriverRepository struct {
	db *gorm.DB
}

// NewDriverRepository builds a DriverRepository backed by db.
func NewDriverRepository(db *gorm.DB) *DriverRepository {
	return &DriverRepository{db: db}
}

// GetByID returns the driver with the given ID, or ErrNotFound.
func (r *DriverRepository) GetByID(ctx context.Context, id int64) (*models.Driver, error) {
	var driver models.Driver
	if err := r.db.WithContext(ctx).First(&driver, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &driver, nil
}

// GetByUberID returns the driver with the given Uber partner ID, or
// ErrNotFound.
func (r *DriverRepository) GetByUberID(ctx context.Context, uberID string) (*models.Driver, error) {
	var driver models.Driver
	if err := r.db.WithContext(ctx).Where("uber_id = ?", uberID).First(&driver).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &driver, nil
}

// Create inserts a new driver.
func (r *DriverRepository) Create(ctx context.Context, driver *models.Driver) error {
	return r.db.WithContext(ctx).Create(driver).Error
}

// Update persists all fields of an existing driver.
func (r *DriverRepository) Update(ctx context.Context, driver *models.Driver) error {
	return r.db.WithContext(ctx).Save(driver).Error
}

// UpsertDriver inserts a driver, or updates the profile/auth fields of an
// existing one matched by uber_id — used by the OAuth login flow, which
// doesn't know ahead of time whether the driver already exists.
func (r *DriverRepository) UpsertDriver(ctx context.Context, driver *models.Driver) error {
	return r.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "uber_id"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"name",
			"email",
			"phone",
			"profile_picture_url",
			"rating",
			"total_trips",
			"account_status",
			"oauth_token",
			"oauth_refresh_token",
			"token_expires_at",
			"last_login_ip",
			"last_login_time",
			"last_synced_at",
		}),
	}).Create(driver).Error
}
