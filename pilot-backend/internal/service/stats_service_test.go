package service

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"pilot-backend/internal/config"
	"pilot-backend/internal/models"
	"pilot-backend/pkg/cache"
	"pilot-backend/pkg/db"
	"pilot-backend/pkg/logger"
)

func testStatsService(t *testing.T) (*StatsService, *gorm.DB, *cache.Redis) {
	t.Helper()

	cfg := config.Load()
	log := logger.New("error")

	gormDB, err := db.Connect(cfg, log)
	if err != nil {
		t.Skipf("mysql not reachable, skipping stats test: %v", err)
	}
	if err := gormDB.AutoMigrate(&models.Driver{}, &models.Trip{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	sqlDB, _ := gormDB.DB()
	t.Cleanup(func() { sqlDB.Close() })

	redisCache := cache.NewRedis(cfg)
	if err := redisCache.Ping(t.Context()); err != nil {
		t.Skipf("redis not reachable, skipping stats test: %v", err)
	}
	t.Cleanup(func() { redisCache.Close() })

	return NewStatsService(gormDB, redisCache, log), gormDB, redisCache
}

func seedStatsDriver(t *testing.T, gormDB *gorm.DB, uberID string) int64 {
	t.Helper()
	driver := &models.Driver{UberID: uberID, Name: "Stats Test", Email: uberID + "@example.com"}
	if err := gormDB.Create(driver).Error; err != nil {
		t.Fatalf("create driver error = %v", err)
	}
	t.Cleanup(func() {
		gormDB.Unscoped().Where("driver_id = ?", driver.ID).Delete(&models.Trip{})
		gormDB.Unscoped().Delete(driver)
	})
	return driver.ID
}

func seedTrip(t *testing.T, gormDB *gorm.DB, driverID int64, uberTripID string, endedAt time.Time, fare, distanceKM float64, status string) {
	t.Helper()
	trip := &models.Trip{
		DriverID:   driverID,
		UberTripID: uberTripID,
		StartedAt:  endedAt.Add(-20 * time.Minute),
		EndedAt:    endedAt,
		DistanceKM: distanceKM,
		FareValue:  fare,
		Status:     status,
	}
	if err := gormDB.Create(trip).Error; err != nil {
		t.Fatalf("create trip error = %v", err)
	}
}

func TestStatsService_Today(t *testing.T) {
	svc, gormDB, _ := testStatsService(t)
	driverID := seedStatsDriver(t, gormDB, "stats-today-"+t.Name())

	now := time.Now()
	seedTrip(t, gormDB, driverID, "today-1-"+t.Name(), now.Add(-2*time.Hour), 30, 10, "COMPLETED")
	seedTrip(t, gormDB, driverID, "today-2-"+t.Name(), now.Add(-1*time.Hour), 20, 5, "COMPLETED")
	// Not counted: cancelled status.
	seedTrip(t, gormDB, driverID, "today-3-"+t.Name(), now.Add(-30*time.Minute), 999, 999, "CANCELLED")

	stats, err := svc.Today(t.Context(), driverID)
	if err != nil {
		t.Fatalf("Today() error = %v", err)
	}

	if stats.TotalTrips != 2 {
		t.Errorf("TotalTrips = %d, want 2 (cancelled trip must be excluded)", stats.TotalTrips)
	}
	if stats.TotalEarned != 50 {
		t.Errorf("TotalEarned = %v, want 50 (30+20)", stats.TotalEarned)
	}
	if stats.AvgFare != 25 {
		t.Errorf("AvgFare = %v, want 25", stats.AvgFare)
	}
	if stats.TotalDistance != 15 {
		t.Errorf("TotalDistance = %v, want 15 (10+5)", stats.TotalDistance)
	}
	if stats.Date != now.Format("2006-01-02") {
		t.Errorf("Date = %q, want %q", stats.Date, now.Format("2006-01-02"))
	}
}

func TestStatsService_Today_NoTrips(t *testing.T) {
	svc, gormDB, _ := testStatsService(t)
	driverID := seedStatsDriver(t, gormDB, "stats-empty-"+t.Name())

	stats, err := svc.Today(t.Context(), driverID)
	if err != nil {
		t.Fatalf("Today() error = %v", err)
	}
	if stats.TotalTrips != 0 || stats.TotalEarned != 0 || stats.EarningsPerHour != 0 {
		t.Errorf("stats = %+v, want all-zero (no division-by-zero NaN/error)", stats)
	}
}

func TestStatsService_Today_UsesCacheOnSecondCall(t *testing.T) {
	svc, gormDB, _ := testStatsService(t)
	driverID := seedStatsDriver(t, gormDB, "stats-cache-"+t.Name())
	seedTrip(t, gormDB, driverID, "cache-1-"+t.Name(), time.Now().Add(-time.Hour), 40, 8, "COMPLETED")

	first, err := svc.Today(t.Context(), driverID)
	if err != nil {
		t.Fatalf("first Today() error = %v", err)
	}
	if first.TotalTrips != 1 {
		t.Fatalf("first TotalTrips = %d, want 1", first.TotalTrips)
	}

	// Insert another trip directly (bypassing the service) after the first
	// call cached the result. If Today() hits the cache, it must NOT see
	// this new trip on the second call within the TTL window.
	seedTrip(t, gormDB, driverID, "cache-2-"+t.Name(), time.Now().Add(-30*time.Minute), 40, 8, "COMPLETED")

	second, err := svc.Today(t.Context(), driverID)
	if err != nil {
		t.Fatalf("second Today() error = %v", err)
	}
	if second.TotalTrips != 1 {
		t.Errorf("second TotalTrips = %d, want 1 (cached — cache functioned as expected)", second.TotalTrips)
	}
}

func TestStatsService_Week(t *testing.T) {
	svc, gormDB, _ := testStatsService(t)
	driverID := seedStatsDriver(t, gormDB, "stats-week-"+t.Name())

	now := time.Now()
	seedTrip(t, gormDB, driverID, "week-1-"+t.Name(), now.Add(-1*24*time.Hour), 100, 10, "COMPLETED")
	seedTrip(t, gormDB, driverID, "week-2-"+t.Name(), now.Add(-2*24*time.Hour), 50, 5, "COMPLETED")
	// Outside the 7-day window.
	seedTrip(t, gormDB, driverID, "week-3-"+t.Name(), now.AddDate(0, 0, -10), 999, 999, "COMPLETED")

	stats, err := svc.Week(t.Context(), driverID)
	if err != nil {
		t.Fatalf("Week() error = %v", err)
	}

	if stats.Summary.TotalTrips != 2 {
		t.Errorf("Summary.TotalTrips = %d, want 2 (10-day-old trip must be excluded)", stats.Summary.TotalTrips)
	}
	if stats.Summary.TotalEarned != 150 {
		t.Errorf("Summary.TotalEarned = %v, want 150 (100+50)", stats.Summary.TotalEarned)
	}
	if len(stats.Days) != 2 {
		t.Errorf("len(Days) = %d, want 2 (one entry per distinct day)", len(stats.Days))
	}
}

func TestStatsService_Month(t *testing.T) {
	svc, gormDB, _ := testStatsService(t)
	driverID := seedStatsDriver(t, gormDB, "stats-month-"+t.Name())

	now := time.Now()
	month := now.Format("2006-01")
	monthStart, _ := time.Parse("2006-01", month)

	seedTrip(t, gormDB, driverID, "month-1-"+t.Name(), monthStart.AddDate(0, 0, 2), 200, 20, "COMPLETED")
	seedTrip(t, gormDB, driverID, "month-2-"+t.Name(), monthStart.AddDate(0, 0, 3), 100, 10, "COMPLETED")
	// Previous month: must be excluded.
	seedTrip(t, gormDB, driverID, "month-3-"+t.Name(), monthStart.AddDate(0, 0, -1), 999, 999, "COMPLETED")

	stats, err := svc.Month(t.Context(), driverID, month)
	if err != nil {
		t.Fatalf("Month() error = %v", err)
	}

	if stats.Summary.TotalTrips != 2 {
		t.Errorf("Summary.TotalTrips = %d, want 2 (previous month's trip must be excluded)", stats.Summary.TotalTrips)
	}
	if stats.Summary.TotalEarned != 300 {
		t.Errorf("Summary.TotalEarned = %v, want 300 (200+100)", stats.Summary.TotalEarned)
	}
	if stats.Month != month {
		t.Errorf("Month = %q, want %q", stats.Month, month)
	}
}

func TestStatsService_Month_InvalidFormat(t *testing.T) {
	svc, gormDB, _ := testStatsService(t)
	driverID := seedStatsDriver(t, gormDB, "stats-month-bad-"+t.Name())

	_, err := svc.Month(t.Context(), driverID, "not-a-month")
	if err == nil {
		t.Fatal("expected an error for an invalid month format, got nil")
	}
}
