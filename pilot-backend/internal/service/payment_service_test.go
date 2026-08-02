package service

import (
	"testing"
	"time"

	"gorm.io/gorm"

	"pilot-backend/internal/models"
	"pilot-backend/internal/oauth"
	"pilot-backend/internal/repository"
)

func testPaymentService(t *testing.T, fake *fakeUberClient) (*PaymentService, models.Driver, *repository.PaymentRepository, *gorm.DB) {
	t.Helper()

	authSvc, driverRepo, gormDB := testAuthService(t, fake)
	if err := gormDB.AutoMigrate(&models.PaymentRecord{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	driver := loggedInDriver(t, authSvc, "code-"+t.Name())
	t.Cleanup(func() {
		gormDB.Unscoped().Where("driver_id = ?", driver.ID).Delete(&models.PaymentRecord{})
		gormDB.Unscoped().Where("id = ?", driver.ID).Delete(&models.Driver{})
	})

	paymentRepo := repository.NewPaymentRepository(gormDB)
	paymentSvc := NewPaymentService(paymentRepo, driverRepo, authSvc.encryptor, fake, authSvc.log)
	return paymentSvc, driver, paymentRepo, gormDB
}

func TestPaymentService_SyncPayments_MapsUberFieldsCorrectly(t *testing.T) {
	fake := &fakeUberClient{}
	paymentSvc, driver, paymentRepo, _ := testPaymentService(t, fake)

	eventTime := time.Now().Add(-time.Hour)
	fake.payments = []oauth.PaymentRecord{
		{
			PaymentID: "pay-" + t.Name(), Category: "fare",
			EventTime: eventTime.Unix(), Amount: 42.50, CurrencyCode: "USD",
			Breakdown: oauth.PaymentBreakdown{Other: 4.16, ServiceFee: -1.04, Toll: 2.50},
		},
	}

	count, _, err := paymentSvc.SyncPayments(t.Context(), driver.ID, 50)
	if err != nil {
		t.Fatalf("SyncPayments() error = %v", err)
	}
	if count != 1 {
		t.Fatalf("count = %d, want 1", count)
	}

	got, err := paymentRepo.GetByUberPaymentID(t.Context(), "pay-"+t.Name())
	if err != nil {
		t.Fatalf("GetByUberPaymentID() error = %v", err)
	}
	if got.Amount != 42.50 {
		t.Errorf("Amount = %v, want 42.50", got.Amount)
	}
	if got.Currency != "USD" {
		t.Errorf("Currency = %q, want USD", got.Currency)
	}
	if got.Tolls != 2.50 {
		t.Errorf("Tolls = %v, want 2.50", got.Tolls)
	}
	if got.Taxes != 1.04 {
		t.Errorf("Taxes = %v, want 1.04 (abs of negative service_fee -1.04)", got.Taxes)
	}
	if got.PaymentMethod != "fare" {
		t.Errorf("PaymentMethod = %q, want fare", got.PaymentMethod)
	}
	wantDate := eventTime.Format("2006-01-02")
	if got.PaymentDate.Format("2006-01-02") != wantDate {
		t.Errorf("PaymentDate = %v, want %v", got.PaymentDate.Format("2006-01-02"), wantDate)
	}
}

func TestPaymentService_SyncPayments_DedupesAlreadySynced(t *testing.T) {
	fake := &fakeUberClient{}
	paymentSvc, driver, _, _ := testPaymentService(t, fake)

	fake.payments = []oauth.PaymentRecord{
		{PaymentID: "dedupe-" + t.Name(), Amount: 10, EventTime: time.Now().Unix()},
	}

	first, _, err := paymentSvc.SyncPayments(t.Context(), driver.ID, 50)
	if err != nil {
		t.Fatalf("first SyncPayments() error = %v", err)
	}
	if first != 1 {
		t.Fatalf("first count = %d, want 1", first)
	}

	second, _, err := paymentSvc.SyncPayments(t.Context(), driver.ID, 50)
	if err != nil {
		t.Fatalf("second SyncPayments() error = %v", err)
	}
	if second != 0 {
		t.Errorf("second count = %d, want 0 (already synced)", second)
	}
}

// TestPaymentService_SyncPayments_EmptyResultIsNotAnError pins the
// fleet-manager scenario documented in uber-integration.md: an empty
// payments list is a valid outcome, not a sync failure.
func TestPaymentService_SyncPayments_EmptyResultIsNotAnError(t *testing.T) {
	fake := &fakeUberClient{payments: nil}
	paymentSvc, driver, _, _ := testPaymentService(t, fake)

	count, syncedAt, err := paymentSvc.SyncPayments(t.Context(), driver.ID, 50)
	if err != nil {
		t.Fatalf("SyncPayments() error = %v, want nil (empty result is valid — e.g. fleet-managed driver)", err)
	}
	if count != 0 {
		t.Errorf("count = %d, want 0", count)
	}
	if syncedAt.IsZero() {
		t.Error("syncedAt is zero")
	}
}

func TestPaymentService_ListPayments(t *testing.T) {
	fake := &fakeUberClient{}
	paymentSvc, driver, paymentRepo, _ := testPaymentService(t, fake)

	today := time.Now()
	payment := &models.PaymentRecord{
		DriverID: driver.ID, UberPaymentID: "list-" + t.Name(),
		Amount: 25.00, PaymentDate: today,
	}
	if err := paymentRepo.Create(t.Context(), payment); err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	payments, pagination, err := paymentSvc.ListPayments(t.Context(), driver.ID, today.Add(-24*time.Hour), today.Add(24*time.Hour), 50, 0)
	if err != nil {
		t.Fatalf("ListPayments() error = %v", err)
	}
	if len(payments) != 1 || pagination.Total != 1 {
		t.Fatalf("payments = %+v, pagination = %+v, want 1 payment", payments, pagination)
	}
}
