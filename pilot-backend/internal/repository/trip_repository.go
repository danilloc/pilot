package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"pilot-backend/internal/models"
)

// TripRepository is the data-access layer for trips.
type TripRepository struct {
	db *gorm.DB
}

// NewTripRepository builds a TripRepository backed by db.
func NewTripRepository(db *gorm.DB) *TripRepository {
	return &TripRepository{db: db}
}

// Create inserts a new trip.
func (r *TripRepository) Create(ctx context.Context, trip *models.Trip) error {
	return r.db.WithContext(ctx).Create(trip).Error
}

// GetByID returns a driver's trip by ID, or ErrNotFound. Scoping by
// driverID enforces data isolation between drivers.
func (r *TripRepository) GetByID(ctx context.Context, driverID, id int64) (*models.Trip, error) {
	var trip models.Trip
	err := r.db.WithContext(ctx).
		Where("id = ? AND driver_id = ?", id, driverID).
		First(&trip).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &trip, nil
}

// GetByUberTripID returns a trip by its Uber trip ID, or ErrNotFound — used
// to dedupe during sync.
func (r *TripRepository) GetByUberTripID(ctx context.Context, uberTripID string) (*models.Trip, error) {
	var trip models.Trip
	err := r.db.WithContext(ctx).Where("uber_trip_id = ?", uberTripID).First(&trip).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &trip, nil
}

// GetByDateRange returns a page of a driver's trips ended within
// [start, end], optionally filtered by status, newest first, along with the
// total count matching the filter (ignoring limit/offset).
func (r *TripRepository) GetByDateRange(ctx context.Context, driverID int64, start, end time.Time, status string, limit, offset int) ([]models.Trip, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.Trip{}).
		Where("driver_id = ? AND ended_at BETWEEN ? AND ?", driverID, start, end)

	if status != "" {
		query = query.Where("status = ?", status)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var trips []models.Trip
	if err := query.Order("ended_at DESC").Limit(limit).Offset(offset).Find(&trips).Error; err != nil {
		return nil, 0, err
	}

	return trips, total, nil
}
