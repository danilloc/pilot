package service

import (
	"errors"
	"testing"

	"pilot-backend/internal/models"
	"pilot-backend/internal/oauth"
)

// loggedInDriver logs a fresh driver in via AuthService (real DB, encrypted
// token stored) so DriverService tests have a realistic starting point.
func loggedInDriver(t *testing.T, authSvc *AuthService, code string) models.Driver {
	t.Helper()
	_, driver, err := authSvc.LoginWithUberCode(t.Context(), code, "127.0.0.1")
	if err != nil {
		t.Fatalf("LoginWithUberCode() error = %v", err)
	}
	return *driver
}

func TestDriverService_GetProfile(t *testing.T) {
	authSvc, driverRepo, gormDB := testAuthService(t, &fakeUberClient{})
	driverSvc := NewDriverService(driverRepo, authSvc.encryptor, authSvc.oauthClient, authSvc.log)

	driver := loggedInDriver(t, authSvc, "code-"+t.Name())
	t.Cleanup(func() { gormDB.Unscoped().Where("id = ?", driver.ID).Delete(&models.Driver{}) })

	got, err := driverSvc.GetProfile(t.Context(), driver.ID)
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}
	if got.ID != driver.ID {
		t.Errorf("ID = %d, want %d", got.ID, driver.ID)
	}
}

func TestDriverService_GetProfile_NotFound(t *testing.T) {
	authSvc, driverRepo, _ := testAuthService(t, &fakeUberClient{})
	driverSvc := NewDriverService(driverRepo, authSvc.encryptor, authSvc.oauthClient, authSvc.log)

	_, err := driverSvc.GetProfile(t.Context(), -1)

	var apiErr *models.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != models.ErrCodeNotFound {
		t.Fatalf("error = %v, want ErrCodeNotFound", err)
	}
}

func TestDriverService_UpdateProfile_Phone(t *testing.T) {
	authSvc, driverRepo, gormDB := testAuthService(t, &fakeUberClient{})
	driverSvc := NewDriverService(driverRepo, authSvc.encryptor, authSvc.oauthClient, authSvc.log)

	driver := loggedInDriver(t, authSvc, "code-"+t.Name())
	t.Cleanup(func() { gormDB.Unscoped().Where("id = ?", driver.ID).Delete(&models.Driver{}) })

	updated, err := driverSvc.UpdateProfile(t.Context(), driver.ID, "+5511988888888")
	if err != nil {
		t.Fatalf("UpdateProfile() error = %v", err)
	}
	if updated.Phone != "+5511988888888" {
		t.Errorf("Phone = %q, want +5511988888888", updated.Phone)
	}
}

func TestDriverService_UpdateProfile_InvalidPhone(t *testing.T) {
	authSvc, driverRepo, gormDB := testAuthService(t, &fakeUberClient{})
	driverSvc := NewDriverService(driverRepo, authSvc.encryptor, authSvc.oauthClient, authSvc.log)

	driver := loggedInDriver(t, authSvc, "code-"+t.Name())
	t.Cleanup(func() { gormDB.Unscoped().Where("id = ?", driver.ID).Delete(&models.Driver{}) })

	_, err := driverSvc.UpdateProfile(t.Context(), driver.ID, "not-a-phone")
	if err != models.ErrInvalidPhone {
		t.Fatalf("error = %v, want ErrInvalidPhone", err)
	}
}

func TestDriverService_SyncProfile(t *testing.T) {
	fake := &fakeUberClient{}
	authSvc, driverRepo, gormDB := testAuthService(t, fake)
	driverSvc := NewDriverService(driverRepo, authSvc.encryptor, fake, authSvc.log)

	driver := loggedInDriver(t, authSvc, "code-"+t.Name())
	t.Cleanup(func() { gormDB.Unscoped().Where("id = ?", driver.ID).Delete(&models.Driver{}) })

	// Change what the fake Uber API returns, to prove sync actually pulls
	// fresh data rather than just touching a timestamp.
	fake.profile = &oauth.Profile{
		DriverID:          driver.UberID,
		FirstName:         "Updated",
		LastName:          "Name",
		Email:             "updated@example.com",
		MobilePhoneNumber: "+5511977777777",
		Rating:            4.5,
	}

	syncedAt, err := driverSvc.SyncProfile(t.Context(), driver.ID)
	if err != nil {
		t.Fatalf("SyncProfile() error = %v", err)
	}
	if syncedAt.IsZero() {
		t.Error("syncedAt is zero")
	}

	got, err := driverRepo.GetByID(t.Context(), driver.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if got.Name != "Updated Name" {
		t.Errorf("Name = %q, want %q", got.Name, "Updated Name")
	}
	if got.Email != "updated@example.com" {
		t.Errorf("Email = %q, want updated@example.com", got.Email)
	}
	if got.LastSyncedAt == nil {
		t.Error("LastSyncedAt was not set")
	}
}
