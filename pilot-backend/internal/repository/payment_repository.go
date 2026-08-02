package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"pilot-backend/internal/models"
)

// PaymentRepository is the data-access layer for payment records.
type PaymentRepository struct {
	db *gorm.DB
}

// NewPaymentRepository builds a PaymentRepository backed by db.
func NewPaymentRepository(db *gorm.DB) *PaymentRepository {
	return &PaymentRepository{db: db}
}

// Create inserts a new payment record.
func (r *PaymentRepository) Create(ctx context.Context, payment *models.PaymentRecord) error {
	return r.db.WithContext(ctx).Create(payment).Error
}

// GetByUberPaymentID returns a payment by its Uber payment ID, or
// ErrNotFound — used to dedupe during sync.
func (r *PaymentRepository) GetByUberPaymentID(ctx context.Context, uberPaymentID string) (*models.PaymentRecord, error) {
	var payment models.PaymentRecord
	err := r.db.WithContext(ctx).Where("uber_payment_id = ?", uberPaymentID).First(&payment).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &payment, nil
}

// GetByDateRange returns a page of a driver's payments within
// [start, end] (inclusive, by payment_date), newest first, along with the
// total count matching the filter (ignoring limit/offset).
func (r *PaymentRepository) GetByDateRange(ctx context.Context, driverID int64, start, end time.Time, limit, offset int) ([]models.PaymentRecord, int64, error) {
	query := r.db.WithContext(ctx).Model(&models.PaymentRecord{}).
		Where("driver_id = ? AND payment_date BETWEEN ? AND ?", driverID, start.Format("2006-01-02"), end.Format("2006-01-02"))

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var payments []models.PaymentRecord
	if err := query.Order("payment_date DESC").Limit(limit).Offset(offset).Find(&payments).Error; err != nil {
		return nil, 0, err
	}

	return payments, total, nil
}
