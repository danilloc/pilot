package repository

import (
	"errors"
	"testing"

	"pilot-backend/internal/models"
)

func TestDriverRepository_CreateAndGetByID(t *testing.T) {
	gormDB := testDB(t)
	repo := NewDriverRepository(gormDB)

	driver := mustCreateDriver(t, gormDB, "uber-create-"+t.Name())

	got, err := repo.GetByID(t.Context(), driver.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Email != driver.Email {
		t.Errorf("Email = %q, want %q", got.Email, driver.Email)
	}
	if got.UUID == "" {
		t.Error("UUID = \"\", want BeforeCreate to have assigned one")
	}
}

func TestDriverRepository_GetByID_NotFound(t *testing.T) {
	gormDB := testDB(t)
	repo := NewDriverRepository(gormDB)

	_, err := repo.GetByID(t.Context(), -1)
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetByID() error = %v, want ErrNotFound", err)
	}
}

func TestDriverRepository_GetByUberID(t *testing.T) {
	gormDB := testDB(t)
	repo := NewDriverRepository(gormDB)

	driver := mustCreateDriver(t, gormDB, "uber-lookup-"+t.Name())

	got, err := repo.GetByUberID(t.Context(), driver.UberID)
	if err != nil {
		t.Fatalf("GetByUberID() error = %v", err)
	}
	if got.ID != driver.ID {
		t.Errorf("ID = %d, want %d", got.ID, driver.ID)
	}
}

func TestDriverRepository_Update(t *testing.T) {
	gormDB := testDB(t)
	repo := NewDriverRepository(gormDB)

	driver := mustCreateDriver(t, gormDB, "uber-update-"+t.Name())
	driver.Name = "Updated Name"

	if err := repo.Update(t.Context(), driver); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	got, err := repo.GetByID(t.Context(), driver.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Name != "Updated Name" {
		t.Errorf("Name = %q, want %q", got.Name, "Updated Name")
	}
}

func TestDriverRepository_UpsertDriver(t *testing.T) {
	gormDB := testDB(t)
	repo := NewDriverRepository(gormDB)
	uberID := "uber-upsert-" + t.Name()

	first := &models.Driver{UberID: uberID, Name: "First", Email: uberID + "-1@example.com"}
	if err := repo.UpsertDriver(t.Context(), first); err != nil {
		t.Fatalf("UpsertDriver() (insert) error = %v", err)
	}
	t.Cleanup(func() { gormDB.Unscoped().Delete(first) })

	second := &models.Driver{UberID: uberID, Name: "Second", Email: uberID + "-1@example.com"}
	if err := repo.UpsertDriver(t.Context(), second); err != nil {
		t.Fatalf("UpsertDriver() (update) error = %v", err)
	}

	got, err := repo.GetByUberID(t.Context(), uberID)
	if err != nil {
		t.Fatalf("GetByUberID() error = %v", err)
	}
	if got.Name != "Second" {
		t.Errorf("Name = %q, want %q (upsert should update, not duplicate)", got.Name, "Second")
	}
}
