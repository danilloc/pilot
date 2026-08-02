package service

import (
	"context"
	"testing"
	"time"

	"golang.org/x/oauth2"
	"gorm.io/gorm"

	"pilot-backend/internal/config"
	"pilot-backend/internal/models"
	"pilot-backend/internal/oauth"
	"pilot-backend/internal/repository"
	"pilot-backend/pkg/cache"
	"pilot-backend/pkg/crypto"
	"pilot-backend/pkg/db"
	"pilot-backend/pkg/jwt"
	"pilot-backend/pkg/logger"
)

const testEncryptionKey = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=" // 32 bytes, base64

// fakeUberClient is a test double for oauth.UberClient: no real network
// calls, fully scriptable success/failure per test case.
type fakeUberClient struct {
	exchangeErr error
	profileErr  error
	profile     *oauth.Profile
	token       *oauth2.Token
}

func (f *fakeUberClient) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	if f.exchangeErr != nil {
		return nil, f.exchangeErr
	}
	if f.token != nil {
		return f.token, nil
	}
	return &oauth2.Token{
		AccessToken:  "access-" + code,
		RefreshToken: "refresh-" + code,
		Expiry:       time.Now().Add(time.Hour),
	}, nil
}

func (f *fakeUberClient) GetProfile(ctx context.Context, token *oauth2.Token) (*oauth.Profile, error) {
	if f.profileErr != nil {
		return nil, f.profileErr
	}
	if f.profile != nil {
		return f.profile, nil
	}
	return &oauth.Profile{
		DriverID:          "uber-" + token.AccessToken,
		FirstName:         "Test",
		LastName:          "Driver",
		Email:             token.AccessToken + "@example.com",
		MobilePhoneNumber: "+5511999999999",
		Rating:            4.9,
	}, nil
}

// testAuthService builds a real AuthService (real MySQL, real Redis, real
// encryptor) wired to a fakeUberClient, so the OAuth boundary is the only
// thing faked. Skips if MySQL/Redis aren't reachable.
func testAuthService(t *testing.T, uberClient oauth.UberClient) (*AuthService, *repository.DriverRepository, *gorm.DB) {
	t.Helper()

	cfg := config.Load()
	log := logger.New("error")

	gormDB, err := db.Connect(cfg, log)
	if err != nil {
		t.Skipf("mysql not reachable, skipping service test: %v", err)
	}
	if err := gormDB.AutoMigrate(&models.Driver{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	sqlDB, _ := gormDB.DB()
	t.Cleanup(func() { sqlDB.Close() })

	redisCache := cache.NewRedis(cfg)
	if err := redisCache.Ping(t.Context()); err != nil {
		t.Skipf("redis not reachable, skipping service test: %v", err)
	}
	t.Cleanup(func() { redisCache.Close() })

	encryptor, err := crypto.NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}

	jwtMgr := jwt.NewManager("test-jwt-secret")
	driverRepo := repository.NewDriverRepository(gormDB)

	svc := NewAuthService(driverRepo, jwtMgr, time.Hour, encryptor, uberClient, redisCache, log)
	return svc, driverRepo, gormDB
}
