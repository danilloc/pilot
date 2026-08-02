package service

import (
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"pilot-backend/internal/config"
	"pilot-backend/internal/models"
	"pilot-backend/internal/repository"
	"pilot-backend/pkg/db"
	"pilot-backend/pkg/logger"
)

func testGoalService(t *testing.T) (*GoalService, *gorm.DB) {
	t.Helper()

	cfg := config.Load()
	log := logger.New("error", "test")

	gormDB, err := db.Connect(cfg, log)
	if err != nil {
		t.Skipf("mysql not reachable, skipping goal test: %v", err)
	}
	if err := gormDB.AutoMigrate(&models.Driver{}, &models.Trip{}, &models.DailyGoal{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	sqlDB, _ := gormDB.DB()
	t.Cleanup(func() { sqlDB.Close() })

	goalRepo := repository.NewGoalRepository(gormDB)
	return NewGoalService(goalRepo, gormDB, log), gormDB
}

func seedGoalDriver(t *testing.T, gormDB *gorm.DB, uberID string) int64 {
	t.Helper()
	driver := &models.Driver{UberID: uberID, Name: "Goal Test", Email: uberID + "@example.com"}
	if err := gormDB.Create(driver).Error; err != nil {
		t.Fatalf("create driver error = %v", err)
	}
	t.Cleanup(func() {
		gormDB.Unscoped().Where("driver_id = ?", driver.ID).Delete(&models.Trip{})
		gormDB.Unscoped().Where("driver_id = ?", driver.ID).Delete(&models.DailyGoal{})
		gormDB.Unscoped().Delete(driver)
	})
	return driver.ID
}

func TestGoalService_Create(t *testing.T) {
	svc, gormDB := testGoalService(t)
	driverID := seedGoalDriver(t, gormDB, "goal-create-"+t.Name())

	goal, err := svc.Create(t.Context(), driverID, time.Now(), 350.00)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if goal.GoalAmount != 350.00 {
		t.Errorf("GoalAmount = %v, want 350.00", goal.GoalAmount)
	}
	if goal.Status != "PENDING" {
		t.Errorf("Status = %q, want PENDING", goal.Status)
	}
}

func TestGoalService_Create_InvalidAmount(t *testing.T) {
	svc, gormDB := testGoalService(t)
	driverID := seedGoalDriver(t, gormDB, "goal-invalid-"+t.Name())

	_, err := svc.Create(t.Context(), driverID, time.Now(), 0)
	if err != models.ErrInvalidGoalAmount {
		t.Fatalf("error = %v, want ErrInvalidGoalAmount", err)
	}
}

func TestGoalService_GetByDate_WithLiveProgress(t *testing.T) {
	svc, gormDB := testGoalService(t)
	driverID := seedGoalDriver(t, gormDB, "goal-progress-"+t.Name())

	today := time.Now()
	if _, err := svc.Create(t.Context(), driverID, today, 100.00); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	trip := &models.Trip{
		DriverID:   driverID,
		UberTripID: "goal-trip-" + t.Name(),
		StartedAt:  today.Add(-time.Hour),
		EndedAt:    today,
		DistanceKM: 10,
		FareValue:  40,
		Status:     "COMPLETED",
	}
	if err := gormDB.Create(trip).Error; err != nil {
		t.Fatalf("create trip error = %v", err)
	}

	got, err := svc.GetByDate(t.Context(), driverID, today)
	if err != nil {
		t.Fatalf("GetByDate() error = %v", err)
	}
	if got.ActualAmount != 40 {
		t.Errorf("ActualAmount = %v, want 40 (live from trips, not stale stored column)", got.ActualAmount)
	}
	if got.PercentComplete != 40 {
		t.Errorf("PercentComplete = %v, want 40 (40/100*100)", got.PercentComplete)
	}
}

func TestGoalService_GetByDate_NotFound(t *testing.T) {
	svc, gormDB := testGoalService(t)
	driverID := seedGoalDriver(t, gormDB, "goal-notfound-"+t.Name())

	_, err := svc.GetByDate(t.Context(), driverID, time.Now().AddDate(0, 0, 30))
	var apiErr *models.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != models.ErrCodeNotFound {
		t.Fatalf("error = %v, want ErrCodeNotFound", err)
	}
}

func TestGoalService_Progress_RemainingNeverNegative(t *testing.T) {
	svc, gormDB := testGoalService(t)
	driverID := seedGoalDriver(t, gormDB, "goal-exceeded-"+t.Name())

	today := time.Now()
	if _, err := svc.Create(t.Context(), driverID, today, 50.00); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	trip := &models.Trip{
		DriverID: driverID, UberTripID: "goal-exceed-trip-" + t.Name(),
		StartedAt: today.Add(-time.Hour), EndedAt: today,
		DistanceKM: 10, FareValue: 100, Status: "COMPLETED",
	}
	if err := gormDB.Create(trip).Error; err != nil {
		t.Fatalf("create trip error = %v", err)
	}

	progress, err := svc.Progress(t.Context(), driverID)
	if err != nil {
		t.Fatalf("Progress() error = %v", err)
	}
	if progress.Remaining != 0 {
		t.Errorf("Remaining = %v, want 0 (goal already exceeded: 100 actual vs 50 goal)", progress.Remaining)
	}
	if progress.ActualAmount != 100 {
		t.Errorf("ActualAmount = %v, want 100", progress.ActualAmount)
	}
}

func TestGoalService_Update(t *testing.T) {
	svc, gormDB := testGoalService(t)
	driverID := seedGoalDriver(t, gormDB, "goal-update-"+t.Name())

	goal, err := svc.Create(t.Context(), driverID, time.Now(), 100.00)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	updated, err := svc.Update(t.Context(), driverID, goal.ID, 200.00)
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.GoalAmount != 200.00 {
		t.Errorf("GoalAmount = %v, want 200.00", updated.GoalAmount)
	}
}

func TestGoalService_Update_WrongDriverNotFound(t *testing.T) {
	svc, gormDB := testGoalService(t)
	owner := seedGoalDriver(t, gormDB, "goal-owner-"+t.Name())
	other := seedGoalDriver(t, gormDB, "goal-other-"+t.Name())

	goal, err := svc.Create(t.Context(), owner, time.Now(), 100.00)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	_, err = svc.Update(t.Context(), other, goal.ID, 999)
	var apiErr *models.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != models.ErrCodeNotFound {
		t.Fatalf("error = %v, want ErrCodeNotFound (data isolation between drivers)", err)
	}
}

func TestGoalService_Abandon(t *testing.T) {
	svc, gormDB := testGoalService(t)
	driverID := seedGoalDriver(t, gormDB, "goal-abandon-"+t.Name())

	goal, err := svc.Create(t.Context(), driverID, time.Now(), 100.00)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := svc.Abandon(t.Context(), driverID, goal.ID); err != nil {
		t.Fatalf("Abandon() error = %v", err)
	}

	updated, err := svc.GetByDate(t.Context(), driverID, goal.GoalDate)
	if err != nil {
		t.Fatalf("GetByDate() error = %v", err)
	}
	if updated.Status != "ABANDONED" {
		t.Errorf("Status = %q, want ABANDONED", updated.Status)
	}
}
