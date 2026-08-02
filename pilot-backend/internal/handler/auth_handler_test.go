package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/oauth2"
	"gorm.io/gorm"

	"pilot-backend/internal/config"
	"pilot-backend/internal/middleware"
	"pilot-backend/internal/models"
	"pilot-backend/internal/oauth"
	"pilot-backend/internal/repository"
	"pilot-backend/internal/service"
	"pilot-backend/pkg/cache"
	"pilot-backend/pkg/crypto"
	"pilot-backend/pkg/db"
	"pilot-backend/pkg/jwt"
	"pilot-backend/pkg/logger"
)

const testEncryptionKey = "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY="

// stubUberClient is a test double for oauth.UberClient: no real network
// calls, deterministic profile derived from the exchanged code.
type stubUberClient struct {
	trips []oauth.TripRecord
}

func (stubUberClient) Exchange(ctx context.Context, code string) (*oauth2.Token, error) {
	return &oauth2.Token{
		AccessToken:  "access-" + code,
		RefreshToken: "refresh-" + code,
		Expiry:       time.Now().Add(time.Hour),
	}, nil
}

func (stubUberClient) GetProfile(ctx context.Context, token *oauth2.Token) (*oauth.Profile, error) {
	return &oauth.Profile{
		DriverID:          "uber-" + token.AccessToken,
		FirstName:         "Test",
		LastName:          "Driver",
		Email:             token.AccessToken + "@example.com",
		MobilePhoneNumber: "+5511999999999",
		Rating:            4.9,
	}, nil
}

func (s stubUberClient) ListTrips(ctx context.Context, token *oauth2.Token, limit int) ([]oauth.TripRecord, error) {
	if len(s.trips) > limit {
		return s.trips[:limit], nil
	}
	return s.trips, nil
}

// testEnv wires a real AuthHandler (real MySQL/Redis, fake Uber) behind a
// real router with the JWT middleware, so these tests exercise the whole
// request path a client would actually hit.
type testEnv struct {
	engine     *gin.Engine
	jwtMgr     *jwt.Manager
	driverRepo *repository.DriverRepository
	uber       *stubUberClient
	gormDB     *gorm.DB
	cleanupIDs []int64
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()

	cfg := config.Load()
	log := logger.New("error")

	gormDB, err := db.Connect(cfg, log)
	if err != nil {
		t.Skipf("mysql not reachable, skipping handler test: %v", err)
	}
	if err := gormDB.AutoMigrate(&models.Driver{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}
	sqlDB, _ := gormDB.DB()
	t.Cleanup(func() { sqlDB.Close() })

	redisCache := cache.NewRedis(cfg)
	if err := redisCache.Ping(t.Context()); err != nil {
		t.Skipf("redis not reachable, skipping handler test: %v", err)
	}
	t.Cleanup(func() { redisCache.Close() })

	encryptor, err := crypto.NewEncryptor(testEncryptionKey)
	if err != nil {
		t.Fatalf("NewEncryptor() error = %v", err)
	}

	if err := gormDB.AutoMigrate(&models.Trip{}); err != nil {
		t.Fatalf("AutoMigrate() error = %v", err)
	}

	jwtMgr := jwt.NewManager("test-jwt-secret")
	driverRepo := repository.NewDriverRepository(gormDB)
	tripRepo := repository.NewTripRepository(gormDB)
	uber := &stubUberClient{}
	authService := service.NewAuthService(driverRepo, jwtMgr, time.Hour, encryptor, uber, redisCache, log)
	authHandler := NewAuthHandler(authService, driverRepo)
	driverService := service.NewDriverService(driverRepo, encryptor, uber, log)
	driverHandler := NewDriverHandler(driverService)
	tripService := service.NewTripService(tripRepo, driverRepo, encryptor, uber, log)
	tripHandler := NewTripHandler(tripService)
	statsService := service.NewStatsService(gormDB, redisCache, log)
	statsHandler := NewStatsHandler(statsService)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(middleware.RequestLogger(log))
	engine.Use(middleware.ErrorHandler(log))
	api := engine.Group("/api")
	authMiddleware := middleware.JWTAuth(jwtMgr, redisCache)
	authHandler.Register(api, authMiddleware)
	driverHandler.Register(api, authMiddleware)
	tripHandler.Register(api, authMiddleware)
	statsHandler.Register(api, authMiddleware)

	env := &testEnv{engine: engine, jwtMgr: jwtMgr, driverRepo: driverRepo, uber: uber, gormDB: gormDB}
	t.Cleanup(func() {
		for _, id := range env.cleanupIDs {
			gormDB.Unscoped().Where("driver_id = ?", id).Delete(&models.Trip{})
			gormDB.Unscoped().Where("id = ?", id).Delete(&models.Driver{})
		}
	})
	return env
}

func (e *testEnv) login(t *testing.T, code string) (string, models.Driver) {
	t.Helper()

	body, _ := json.Marshal(map[string]string{"code": code})
	req := httptest.NewRequest(http.MethodPost, "/api/auth/uber-login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	e.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("uber-login status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Token  string        `json:"token"`
		Driver models.Driver `json:"driver"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	e.cleanupIDs = append(e.cleanupIDs, resp.Driver.ID)
	return resp.Token, resp.Driver
}

func TestAuthHandler_UberLogin_MissingCode(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodPost, "/api/auth/uber-login", bytes.NewReader([]byte(`{}`)))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest && rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 400 or 401 for missing code", rec.Code)
	}
}

func TestAuthHandler_UberLogin_Success(t *testing.T) {
	env := newTestEnv(t)

	token, driver := env.login(t, "code-"+t.Name())

	if token == "" {
		t.Error("token is empty")
	}
	if driver.Email == "" {
		t.Error("driver.Email is empty")
	}
}

func TestAuthHandler_Me_RequiresAuth(t *testing.T) {
	env := newTestEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestAuthHandler_Me_ReturnsAuthenticatedDriver(t *testing.T) {
	env := newTestEnv(t)
	token, driver := env.login(t, "code-"+t.Name())

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	env.engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	var got models.Driver
	if err := json.Unmarshal(rec.Body.Bytes(), &got); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if got.ID != driver.ID {
		t.Errorf("ID = %d, want %d", got.ID, driver.ID)
	}
}

func TestAuthHandler_Logout_ThenTokenRejected(t *testing.T) {
	env := newTestEnv(t)
	token, _ := env.login(t, "code-"+t.Name())

	logoutReq := httptest.NewRequest(http.MethodPost, "/api/auth/logout", nil)
	logoutReq.Header.Set("Authorization", "Bearer "+token)
	logoutRec := httptest.NewRecorder()
	env.engine.ServeHTTP(logoutRec, logoutReq)

	if logoutRec.Code != http.StatusOK {
		t.Fatalf("logout status = %d, body = %s", logoutRec.Code, logoutRec.Body.String())
	}

	meReq := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meReq.Header.Set("Authorization", "Bearer "+token)
	meRec := httptest.NewRecorder()
	env.engine.ServeHTTP(meRec, meReq)

	if meRec.Code != http.StatusUnauthorized {
		t.Fatalf("post-logout /me status = %d, want 401 (token should be blacklisted)", meRec.Code)
	}
}
