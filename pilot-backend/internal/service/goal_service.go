package service

import (
	"context"
	"time"

	"gorm.io/gorm"

	"pilot-backend/internal/models"
	"pilot-backend/internal/repository"
	"pilot-backend/pkg/logger"
)

// GoalWithProgress is a daily goal combined with its live progress,
// computed from the driver's completed trips on that date (not the
// possibly-stale daily_goals.actual_amount column).
type GoalWithProgress struct {
	ID              int64   `json:"id"`
	UUID            string  `json:"uuid"`
	DriverID        int64   `json:"driver_id"`
	GoalDate        string  `json:"goal_date"`
	GoalAmount      float64 `json:"goal_amount"`
	ActualAmount    float64 `json:"actual_amount"`
	PercentComplete float64 `json:"percent_complete"`
	Status          string  `json:"status"`
}

// GoalProgress is the response shape for GET /api/goals/progress.
type GoalProgress struct {
	GoalAmount      float64 `json:"goal_amount"`
	ActualAmount    float64 `json:"actual_amount"`
	PercentComplete float64 `json:"percent_complete"`
	Remaining       float64 `json:"remaining"`
}

// GoalService implements daily goal CRUD and progress tracking against a
// driver's completed trips.
type GoalService struct {
	goalRepo *repository.GoalRepository
	db       *gorm.DB
	log      *logger.Logger
}

// NewGoalService builds a GoalService from its collaborators.
func NewGoalService(goalRepo *repository.GoalRepository, db *gorm.DB, log *logger.Logger) *GoalService {
	return &GoalService{goalRepo: goalRepo, db: db, log: log}
}

// Create adds a new daily goal for driverID on goalDate.
func (s *GoalService) Create(ctx context.Context, driverID int64, goalDate time.Time, goalAmount float64) (*models.DailyGoal, error) {
	goal := &models.DailyGoal{
		DriverID: driverID,
		// Re-anchor to time.Local at midnight before it goes through GORM:
		// the MySQL DSN uses loc=Local, so the driver converts any time.Time
		// parameter to Local before writing a DATE column. A caller-supplied
		// date parsed with time.Parse (which defaults to UTC) would then
		// shift by the local UTC offset and land on the wrong calendar day
		// (e.g. UTC-3 turns 2026-06-01 00:00 UTC into 2026-05-31 21:00
		// local). Normalizing here makes the Local conversion a no-op.
		GoalDate:   time.Date(goalDate.Year(), goalDate.Month(), goalDate.Day(), 0, 0, 0, 0, time.Local),
		GoalAmount: goalAmount,
		Status:     "PENDING",
	}
	if err := goal.Validate(); err != nil {
		return nil, err
	}
	if err := s.goalRepo.Create(ctx, goal); err != nil {
		s.log.Errorw("goal.create_failed", "driver_id", driverID, "error", err.Error())
		return nil, models.NewAPIError(models.ErrCodeDatabaseError, "failed to create goal")
	}
	return goal, nil
}

// GetByDate returns driverID's goal for date, combined with live progress.
func (s *GoalService) GetByDate(ctx context.Context, driverID int64, date time.Time) (*GoalWithProgress, error) {
	goal, err := s.goalRepo.GetByDate(ctx, driverID, date)
	if err != nil {
		return nil, mapRepoErr(err)
	}
	return s.withProgress(ctx, goal)
}

// Progress returns driverID's progress toward today's goal.
func (s *GoalService) Progress(ctx context.Context, driverID int64) (*GoalProgress, error) {
	goal, err := s.goalRepo.GetByDate(ctx, driverID, time.Now())
	if err != nil {
		return nil, mapRepoErr(err)
	}

	withProgress, err := s.withProgress(ctx, goal)
	if err != nil {
		return nil, err
	}

	remaining := withProgress.GoalAmount - withProgress.ActualAmount
	if remaining < 0 {
		remaining = 0
	}

	return &GoalProgress{
		GoalAmount:      withProgress.GoalAmount,
		ActualAmount:    withProgress.ActualAmount,
		PercentComplete: withProgress.PercentComplete,
		Remaining:       remaining,
	}, nil
}

// Update changes goalID's target amount, scoped to driverID.
func (s *GoalService) Update(ctx context.Context, driverID, goalID int64, goalAmount float64) (*models.DailyGoal, error) {
	goal, err := s.goalRepo.GetByID(ctx, driverID, goalID)
	if err != nil {
		return nil, mapRepoErr(err)
	}

	goal.GoalAmount = goalAmount
	if err := goal.Validate(); err != nil {
		return nil, err
	}

	if err := s.goalRepo.Update(ctx, goal); err != nil {
		s.log.Errorw("goal.update_failed", "driver_id", driverID, "goal_id", goalID, "error", err.Error())
		return nil, models.NewAPIError(models.ErrCodeDatabaseError, "failed to update goal")
	}
	return goal, nil
}

// Abandon marks goalID as ABANDONED, scoped to driverID. Goals are never
// hard-deleted — design.md's DELETE response confirms the resulting
// status rather than a deletion.
func (s *GoalService) Abandon(ctx context.Context, driverID, goalID int64) error {
	goal, err := s.goalRepo.GetByID(ctx, driverID, goalID)
	if err != nil {
		return mapRepoErr(err)
	}

	goal.Status = "ABANDONED"
	if err := s.goalRepo.Update(ctx, goal); err != nil {
		s.log.Errorw("goal.abandon_failed", "driver_id", driverID, "goal_id", goalID, "error", err.Error())
		return models.NewAPIError(models.ErrCodeDatabaseError, "failed to abandon goal")
	}
	return nil
}

type goalProgressRow struct {
	ActualAmount float64
	Percent      float64
}

// withProgress computes a goal's live progress from completed trips ended
// on the goal's date, using the same sargable half-open date-range join
// confirmed in the database feature's goal_progress query.
func (s *GoalService) withProgress(ctx context.Context, goal *models.DailyGoal) (*GoalWithProgress, error) {
	var row goalProgressRow
	err := s.db.WithContext(ctx).Raw(`
		SELECT
			COALESCE(SUM(t.fare_value), 0) as actual_amount,
			(COALESCE(SUM(t.fare_value), 0) / ? * 100) as percent
		FROM trips t
		WHERE t.driver_id = ?
		  AND t.ended_at >= ?
		  AND t.ended_at < ? + INTERVAL 1 DAY
		  AND t.status = 'COMPLETED'`,
		goal.GoalAmount, goal.DriverID, goal.GoalDate, goal.GoalDate,
	).Scan(&row).Error
	if err != nil {
		s.log.Errorw("goal.progress_query_failed", "driver_id", goal.DriverID, "goal_id", goal.ID, "error", err.Error())
		return nil, models.NewAPIError(models.ErrCodeDatabaseError, "database error")
	}

	return &GoalWithProgress{
		ID:              goal.ID,
		UUID:            goal.UUID,
		DriverID:        goal.DriverID,
		GoalDate:        goal.GoalDate.Format("2006-01-02"),
		GoalAmount:      goal.GoalAmount,
		ActualAmount:    row.ActualAmount,
		PercentComplete: row.Percent,
		Status:          goal.Status,
	}, nil
}
