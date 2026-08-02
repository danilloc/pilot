package repository

import (
	"testing"
	"time"

	"pilot-backend/internal/models"
)

func TestPaymentRepository_CreateAndGetByDateRange(t *testing.T) {
	gormDB := testDB(t)
	driver := mustCreateDriver(t, gormDB, "uber-payment-"+t.Name())
	repo := NewPaymentRepository(gormDB)

	today := time.Now().Truncate(24 * time.Hour)
	payment := &models.PaymentRecord{
		DriverID:    driver.ID,
		Amount:      120.50,
		PaymentDate: today,
	}
	if err := repo.Create(t.Context(), payment); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	payments, total, err := repo.GetByDateRange(t.Context(), driver.ID, today.Add(-24*time.Hour), today.Add(24*time.Hour), 50, 0)
	if err != nil {
		t.Fatalf("GetByDateRange() error = %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(payments) != 1 || payments[0].Amount != 120.50 {
		t.Errorf("payments = %+v, want one record with Amount=120.50", payments)
	}
}
