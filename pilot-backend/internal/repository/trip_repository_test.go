package repository

import (
	"errors"
	"testing"
	"time"

	"pilot-backend/internal/models"
)

func TestTripRepository_CreateAndGetByID(t *testing.T) {
	gormDB := testDB(t)
	driver := mustCreateDriver(t, gormDB, "uber-trip-create-"+t.Name())
	repo := NewTripRepository(gormDB)

	trip := &models.Trip{
		DriverID:   driver.ID,
		UberTripID: "trip-" + t.Name(),
		StartedAt:  time.Now().Add(-30 * time.Minute),
		EndedAt:    time.Now(),
		DistanceKM: 12.5,
		FareValue:  45.90,
	}
	if err := repo.Create(t.Context(), trip); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	got, err := repo.GetByID(t.Context(), driver.ID, trip.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.FareValue != trip.FareValue {
		t.Errorf("FareValue = %v, want %v", got.FareValue, trip.FareValue)
	}
}

func TestTripRepository_GetByID_WrongDriverIsolated(t *testing.T) {
	gormDB := testDB(t)
	owner := mustCreateDriver(t, gormDB, "uber-trip-owner-"+t.Name())
	other := mustCreateDriver(t, gormDB, "uber-trip-other-"+t.Name())
	repo := NewTripRepository(gormDB)

	trip := &models.Trip{
		DriverID:   owner.ID,
		UberTripID: "trip-isolation-" + t.Name(),
		StartedAt:  time.Now().Add(-time.Hour),
		EndedAt:    time.Now(),
		DistanceKM: 5,
		FareValue:  20,
	}
	if err := repo.Create(t.Context(), trip); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if _, err := repo.GetByID(t.Context(), other.ID, trip.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() with wrong driver error = %v, want ErrNotFound", err)
	}
}

func TestTripRepository_GetByDateRange(t *testing.T) {
	gormDB := testDB(t)
	driver := mustCreateDriver(t, gormDB, "uber-trip-range-"+t.Name())
	repo := NewTripRepository(gormDB)

	now := time.Now()
	for i := 0; i < 3; i++ {
		trip := &models.Trip{
			DriverID:   driver.ID,
			UberTripID: "trip-range-" + t.Name() + "-" + time.Duration(i).String(),
			StartedAt:  now.Add(-time.Duration(i+1) * time.Hour),
			EndedAt:    now.Add(-time.Duration(i) * time.Hour),
			DistanceKM: 10,
			FareValue:  30,
			Status:     "COMPLETED",
		}
		if err := repo.Create(t.Context(), trip); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
	}

	trips, total, err := repo.GetByDateRange(t.Context(), driver.ID, now.Add(-24*time.Hour), now.Add(time.Hour), "COMPLETED", 2, 0)
	if err != nil {
		t.Fatalf("GetByDateRange() error = %v", err)
	}
	if total != 3 {
		t.Errorf("total = %d, want 3", total)
	}
	if len(trips) != 2 {
		t.Errorf("len(trips) = %d, want 2 (limit)", len(trips))
	}
}
