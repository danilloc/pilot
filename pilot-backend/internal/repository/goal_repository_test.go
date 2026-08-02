package repository

import (
	"testing"
	"time"

	"pilot-backend/internal/models"
)

func TestGoalRepository_CreateAndGetByDate(t *testing.T) {
	gormDB := testDB(t)
	driver := mustCreateDriver(t, gormDB, "uber-goal-create-"+t.Name())
	repo := NewGoalRepository(gormDB)

	today := time.Now().Truncate(24 * time.Hour)
	goal := &models.DailyGoal{
		DriverID:   driver.ID,
		GoalDate:   today,
		GoalAmount: 350.00,
	}
	if err := repo.Create(t.Context(), goal); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.GetByDate(t.Context(), driver.ID, today)
	if err != nil {
		t.Fatalf("GetByDate() error = %v", err)
	}
	if got.GoalAmount != 350.00 {
		t.Errorf("GoalAmount = %v, want 350.00", got.GoalAmount)
	}
}

func TestGoalRepository_Update(t *testing.T) {
	gormDB := testDB(t)
	driver := mustCreateDriver(t, gormDB, "uber-goal-update-"+t.Name())
	repo := NewGoalRepository(gormDB)

	today := time.Now().Truncate(24 * time.Hour)
	goal := &models.DailyGoal{DriverID: driver.ID, GoalDate: today, GoalAmount: 300.00}
	if err := repo.Create(t.Context(), goal); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	goal.GoalAmount = 400.00
	if err := repo.Update(t.Context(), goal); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := repo.GetByDate(t.Context(), driver.ID, today)
	if err != nil {
		t.Fatalf("GetByDate() error = %v", err)
	}
	if got.GoalAmount != 400.00 {
		t.Errorf("GoalAmount = %v, want 400.00", got.GoalAmount)
	}
}
