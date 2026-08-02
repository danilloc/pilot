package service

import (
	"errors"
	"testing"

	"pilot-backend/internal/models"
)

func TestAuthService_LoginWithUberCode_Success(t *testing.T) {
	fake := &fakeUberClient{}
	svc, _, gormDB := testAuthService(t, fake)

	token, driver, err := svc.LoginWithUberCode(t.Context(), "code-"+t.Name(), "127.0.0.1")
	if err != nil {
		t.Fatalf("LoginWithUberCode() error = %v", err)
	}
	t.Cleanup(func() { gormDB.Unscoped().Where("id = ?", driver.ID).Delete(&models.Driver{}) })

	if token == "" {
		t.Error("token is empty")
	}
	if driver.ID == 0 {
		t.Error("driver.ID = 0, want a persisted ID (re-fetch after upsert should populate it)")
	}
	if driver.OAuthToken != "" || driver.OAuthRefreshToken != "" {
		t.Error("LoginWithUberCode returned a driver with tokens not redacted")
	}
}

func TestAuthService_LoginWithUberCode_EmptyCode(t *testing.T) {
	svc, _, _ := testAuthService(t, &fakeUberClient{})

	_, _, err := svc.LoginWithUberCode(t.Context(), "", "127.0.0.1")

	var apiErr *models.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != models.ErrCodeOAuthFailed {
		t.Fatalf("error = %v, want ErrCodeOAuthFailed", err)
	}
}

func TestAuthService_LoginWithUberCode_ExchangeFails(t *testing.T) {
	fake := &fakeUberClient{exchangeErr: errors.New("uber is down")}
	svc, _, _ := testAuthService(t, fake)

	_, _, err := svc.LoginWithUberCode(t.Context(), "code-"+t.Name(), "127.0.0.1")

	var apiErr *models.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != models.ErrCodeOAuthFailed {
		t.Fatalf("error = %v, want ErrCodeOAuthFailed", err)
	}
}

func TestAuthService_LoginWithUberCode_UpsertsExistingDriver(t *testing.T) {
	fake := &fakeUberClient{}
	svc, driverRepo, gormDB := testAuthService(t, fake)

	code := "code-" + t.Name()

	_, first, err := svc.LoginWithUberCode(t.Context(), code, "127.0.0.1")
	if err != nil {
		t.Fatalf("first login: LoginWithUberCode() error = %v", err)
	}
	t.Cleanup(func() { gormDB.Unscoped().Where("id = ?", first.ID).Delete(&models.Driver{}) })

	_, second, err := svc.LoginWithUberCode(t.Context(), code, "127.0.0.1")
	if err != nil {
		t.Fatalf("second login: LoginWithUberCode() error = %v", err)
	}

	if second.ID != first.ID {
		t.Errorf("second login ID = %d, want %d (same driver, not a duplicate)", second.ID, first.ID)
	}

	stored, err := driverRepo.GetByUberID(t.Context(), first.UberID)
	if err != nil {
		t.Fatalf("GetByUberID() error = %v", err)
	}
	if stored.ID != first.ID {
		t.Errorf("stored driver ID = %d, want %d", stored.ID, first.ID)
	}
}

func TestAuthService_LoginWithUberCode_BannedDriverRejected(t *testing.T) {
	fake := &fakeUberClient{}
	svc, driverRepo, gormDB := testAuthService(t, fake)

	code := "code-" + t.Name()

	_, driver, err := svc.LoginWithUberCode(t.Context(), code, "127.0.0.1")
	if err != nil {
		t.Fatalf("first login: LoginWithUberCode() error = %v", err)
	}
	t.Cleanup(func() { gormDB.Unscoped().Where("id = ?", driver.ID).Delete(&models.Driver{}) })

	persisted, err := driverRepo.GetByID(t.Context(), driver.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	persisted.IsBanned = true
	if err := driverRepo.Update(t.Context(), persisted); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	_, _, err = svc.LoginWithUberCode(t.Context(), code, "127.0.0.1")
	if err != models.ErrDriverBanned {
		t.Fatalf("error = %v, want ErrDriverBanned", err)
	}
}

func TestAuthService_Logout(t *testing.T) {
	fake := &fakeUberClient{}
	svc, _, gormDB := testAuthService(t, fake)

	token, driver, err := svc.LoginWithUberCode(t.Context(), "code-"+t.Name(), "127.0.0.1")
	if err != nil {
		t.Fatalf("LoginWithUberCode() error = %v", err)
	}
	t.Cleanup(func() { gormDB.Unscoped().Where("id = ?", driver.ID).Delete(&models.Driver{}) })

	if err := svc.Logout(t.Context(), token); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
}

func TestAuthService_Logout_MalformedToken(t *testing.T) {
	svc, _, _ := testAuthService(t, &fakeUberClient{})

	err := svc.Logout(t.Context(), "not-a-real-token")

	var apiErr *models.APIError
	if !errors.As(err, &apiErr) || apiErr.Code != models.ErrCodeInvalidToken {
		t.Fatalf("error = %v, want ErrCodeInvalidToken", err)
	}
}
