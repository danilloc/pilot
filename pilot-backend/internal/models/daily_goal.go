package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// DailyGoal mirrors the daily_goals table. A driver has at most one goal per
// calendar date (unique_driver_date).
type DailyGoal struct {
	ID       int64  `gorm:"primaryKey;autoIncrement" json:"id"`
	UUID     string `gorm:"type:varchar(36);uniqueIndex;not null" json:"uuid"`
	DriverID int64  `gorm:"not null;uniqueIndex:unique_driver_date" json:"driver_id"`

	GoalDate     time.Time `gorm:"type:date;not null;uniqueIndex:unique_driver_date" json:"goal_date"`
	GoalAmount   float64   `gorm:"type:decimal(10,2);not null" json:"goal_amount"`
	ActualAmount float64   `gorm:"type:decimal(10,2);default:0" json:"actual_amount"`
	Status       string    `gorm:"type:varchar(50);default:PENDING" json:"status"`

	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

func (DailyGoal) TableName() string { return "daily_goals" }

// BeforeCreate assigns a public UUID when one hasn't already been set.
func (g *DailyGoal) BeforeCreate(tx *gorm.DB) error {
	if g.UUID == "" {
		g.UUID = uuid.NewString()
	}
	return nil
}

// Validate checks the business-level invariants required before a
// DailyGoal can be persisted.
func (g *DailyGoal) Validate() error {
	if g.GoalAmount <= 0 {
		return ErrInvalidGoalAmount
	}
	return nil
}
