package repository

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	"pilot-backend/internal/models"
)

// GoalRepository is the data-access layer for daily goals.
type GoalRepository struct {
	db *gorm.DB
}

// NewGoalRepository builds a GoalRepository backed by db.
func NewGoalRepository(db *gorm.DB) *GoalRepository {
	return &GoalRepository{db: db}
}

// Create inserts a new daily goal.
func (r *GoalRepository) Create(ctx context.Context, goal *models.DailyGoal) error {
	return r.db.WithContext(ctx).Create(goal).Error
}

// GetByDate returns a driver's goal for the given date, or ErrNotFound.
func (r *GoalRepository) GetByDate(ctx context.Context, driverID int64, date time.Time) (*models.DailyGoal, error) {
	var goal models.DailyGoal
	err := r.db.WithContext(ctx).
		Where("driver_id = ? AND goal_date = ?", driverID, date.Format("2006-01-02")).
		First(&goal).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &goal, nil
}

// Update persists all fields of an existing goal.
func (r *GoalRepository) Update(ctx context.Context, goal *models.DailyGoal) error {
	return r.db.WithContext(ctx).Save(goal).Error
}
