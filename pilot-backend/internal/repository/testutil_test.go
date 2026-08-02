package repository

import (
	"testing"

	"gorm.io/gorm"

	"pilot-backend/internal/config"
	"pilot-backend/internal/models"
	"pilot-backend/pkg/db"
	"pilot-backend/pkg/logger"
)

// testDB connects to the MySQL instance described by the environment (see
// docker-compose.yml) and ensures the tables under test exist. It skips the
// calling test when no database is reachable, e.g. outside Docker.
func testDB(t *testing.T) *gorm.DB {
	t.Helper()

	cfg := config.Load()
	log := logger.New("error", "test")

	gormDB, err := db.Connect(cfg, log)
	if err != nil {
		t.Skipf("mysql not reachable, skipping repository test: %v", err)
	}

	if err := gormDB.AutoMigrate(
		&models.Driver{},
		&models.Trip{},
		&models.DailyGoal{},
		&models.PaymentRecord{},
	); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	return gormDB
}

// mustCreateDriver inserts a throwaway driver for FK-dependent tests and
// registers its (cascading) cleanup.
func mustCreateDriver(t *testing.T, gormDB *gorm.DB, uberID string) *models.Driver {
	t.Helper()

	driver := &models.Driver{
		UberID: uberID,
		Name:   "Test Driver",
		Email:  uberID + "@example.com",
	}
	if err := NewDriverRepository(gormDB).Create(t.Context(), driver); err != nil {
		t.Fatalf("Create(driver) error = %v", err)
	}
	t.Cleanup(func() {
		// AutoMigrate doesn't set up real FK cascade constraints (the models
		// declare no Go-level association), so child rows are cleaned up
		// explicitly rather than relying on ON DELETE CASCADE.
		gormDB.Unscoped().Where("driver_id = ?", driver.ID).Delete(&models.Trip{})
		gormDB.Unscoped().Where("driver_id = ?", driver.ID).Delete(&models.DailyGoal{})
		gormDB.Unscoped().Where("driver_id = ?", driver.ID).Delete(&models.PaymentRecord{})
		gormDB.Unscoped().Delete(driver)
	})
	return driver
}
