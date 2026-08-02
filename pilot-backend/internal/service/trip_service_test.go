package service

import (
	"errors"
	"testing"
	"time"

	"gorm.io/gorm"

	"pilot-backend/internal/models"
	"pilot-backend/internal/oauth"
	"pilot-backend/internal/repository"
)

// testTripService wires a real TripService on top of the same real
// MySQL/Redis stack testAuthService uses, plus a logged-in driver to own
// the trips under test.
func testTripService(t *testing.T, fake *fakeUberClient) (*TripService, models.Driver, *repository.TripRepository, *gorm.DB) {
	t.Helper()

	authSvc, driverRepo, gormDB := testAuthService(t, fake)
	if err := gormDB.AutoMigrate(&models.Trip{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	driver := loggedInDriver(t, authSvc, "code-"+t.Name())
	t.Cleanup(func() {
		gormDB.Unscoped().Where("driver_id = ?", driver.ID).Delete(&models.Trip{})
		gormDB.Unscoped().Where("id = ?", driver.ID).Delete(&models.Driver{})
	})

	tripRepo := repository.NewTripRepository(gormDB)
	tripSvc := NewTripService(tripRepo, driverRepo, authSvc.encryptor, fake, authSvc.log)
	return tripSvc, driver, tripRepo, gormDB
}

func TestTripService_SyncTrips_PersistsNewTrips(t *testing.T) {
	fake := &fakeUberClient{}
	tripSvc, driver, _, _ := testTripService(t, fake)

	now := time.Now()
	fake.trips = []oauth.TripRecord{
		{TripID: "sync-" + t.Name() + "-1", StartTime: now.Add(-time.Hour).Unix(), EndTime: now.Unix(), DistanceKM: 10, Fare: 30},
		{TripID: "sync-" + t.Name() + "-2", StartTime: now.Add(-2 * time.Hour).Unix(), EndTime: now.Add(-time.Hour).Unix(), DistanceKM: 5, Fare: 15},
	}

	count, syncedAt, err := tripSvc.SyncTrips(t.Context(), driver.ID, 50)
	if err != nil {
		t.Fatalf("SyncTrips() error = %v", err)
	}
	if count != 2 {
		t.Errorf("count = %d, want 2", count)
	}
	if syncedAt.IsZero() {
		t.Error("syncedAt is zero")
	}
}

func TestTripService_SyncTrips_DedupesAlreadySynced(t *testing.T) {
	fake := &fakeUberClient{}
	tripSvc, driver, _, _ := testTripService(t, fake)

	now := time.Now()
	fake.trips = []oauth.TripRecord{
		{TripID: "dedupe-" + t.Name(), StartTime: now.Add(-time.Hour).Unix(), EndTime: now.Unix(), DistanceKM: 10, Fare: 30},
	}

	first, _, err := tripSvc.SyncTrips(t.Context(), driver.ID, 50)
	if err != nil {
		t.Fatalf("first SyncTrips() error = %v", err)
	}
	if first != 1 {
		t.Fatalf("first count = %d, want 1", first)
	}

	second, _, err := tripSvc.SyncTrips(t.Context(), driver.ID, 50)
	if err != nil {
		t.Fatalf("second SyncTrips() error = %v", err)
	}
	if second != 0 {
		t.Errorf("second count = %d, want 0 (already synced)", second)
	}
}

func TestTripService_SyncTrips_SkipsInvalidRecords(t *testing.T) {
	fake := &fakeUberClient{}
	tripSvc, driver, _, _ := testTripService(t, fake)

	now := time.Now()
	fake.trips = []oauth.TripRecord{
		// Invalid: zero distance.
		{TripID: "invalid-" + t.Name(), StartTime: now.Add(-time.Hour).Unix(), EndTime: now.Unix(), DistanceKM: 0, Fare: 30},
	}

	count, _, err := tripSvc.SyncTrips(t.Context(), driver.ID, 50)
	if err != nil {
		t.Fatalf("SyncTrips() error = %v", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0 (invalid record should be skipped, not fail the whole sync)", count)
	}
}

func TestTripService_ListAndGetTrip(t *testing.T) {
	fake := &fakeUberClient{}
	tripSvc, driver, tripRepo, _ := testTripService(t, fake)

	now := time.Now()
	trip := &models.Trip{
		DriverID:   driver.ID,
		UberTripID: "list-" + t.Name(),
		StartedAt:  now.Add(-time.Hour),
		EndedAt:    now,
		DistanceKM: 8,
		FareValue:  40,
		Status:     "COMPLETED",
	}
	if err := tripRepo.Create(t.Context(), trip); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	trips, pagination, err := tripSvc.ListTrips(t.Context(), driver.ID, now.Add(-24*time.Hour), now.Add(time.Hour), "COMPLETED", 10, 0)
	if err != nil {
		t.Fatalf("ListTrips() error = %v", err)
	}
	if len(trips) != 1 || pagination.Total != 1 {
		t.Fatalf("trips = %+v, pagination = %+v, want 1 trip", trips, pagination)
	}

	got, err := tripSvc.GetTrip(t.Context(), driver.ID, trip.ID)
	if err != nil {
		t.Fatalf("GetTrip() error = %v", err)
	}
	if got.ID != trip.ID {
		t.Errorf("ID = %d, want %d", got.ID, trip.ID)
	}
}

func TestTripService_GetTrip_WrongDriverNotFound(t *testing.T) {
	fake := &fakeUberClient{}
	tripSvc, driver, tripRepo, gormDB := testTripService(t, fake)

	other := mustCreateOtherDriver(t, gormDB)

	trip := &models.Trip{
		DriverID:   other.ID,
		UberTripID: "isolated-" + t.Name(),
		StartedAt:  time.Now().Add(-time.Hour),
		EndedAt:    time.Now(),
		DistanceKM: 5,
		FareValue:  10,
	}
	if err := tripRepo.Create(t.Context(), trip); err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	t.Cleanup(func() {
		gormDB.Unscoped().Where("driver_id = ?", other.ID).Delete(&models.Trip{})
		gormDB.Unscoped().Where("id = ?", other.ID).Delete(&models.Driver{})
	})

	_, err := tripSvc.GetTrip(t.Context(), driver.ID, trip.ID)
	var apiErr *models.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != models.ErrCodeNotFound {
		t.Fatalf("error = %v, want ErrCodeNotFound (data isolation between drivers)", err)
	}
}

func mustCreateOtherDriver(t *testing.T, gormDB *gorm.DB) *models.Driver {
	t.Helper()
	driver := &models.Driver{UberID: "other-" + t.Name(), Name: "Other", Email: "other-" + t.Name() + "@example.com"}
	if err := gormDB.Create(driver).Error; err != nil {
		t.Fatalf("Create(other driver) error = %v", err)
	}
	return driver
}
