package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"pilot-backend/pkg/jwt"
)

func setupAuthEngine(t *testing.T, mgr *jwt.Manager) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	redisCache := testRedis(t)

	engine := gin.New()
	engine.Use(JWTAuth(mgr, redisCache))
	engine.GET("/protected", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"driver_id": c.GetInt64("driver_id")})
	})
	return engine
}

func TestJWTAuth_MissingToken(t *testing.T) {
	mgr := jwt.NewManager("secret")
	engine := setupAuthEngine(t, mgr)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestJWTAuth_InvalidFormat(t *testing.T) {
	mgr := jwt.NewManager("secret")
	engine := setupAuthEngine(t, mgr)

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "NotBearer sometoken")
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestJWTAuth_ValidToken(t *testing.T) {
	mgr := jwt.NewManager("secret")
	engine := setupAuthEngine(t, mgr)

	token, err := mgr.GenerateToken(99, "d@example.com", time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}
}

func TestJWTAuth_ExpiredToken(t *testing.T) {
	mgr := jwt.NewManager("secret")
	engine := setupAuthEngine(t, mgr)

	token, err := mgr.GenerateToken(1, "d@example.com", -time.Minute)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}

func TestJWTAuth_BlacklistedToken(t *testing.T) {
	mgr := jwt.NewManager("secret")
	redisCache := testRedis(t)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(JWTAuth(mgr, redisCache))
	engine.GET("/protected", func(c *gin.Context) { c.Status(http.StatusOK) })

	token, err := mgr.GenerateToken(1, "d@example.com", time.Hour)
	if err != nil {
		t.Fatalf("GenerateToken() error = %v", err)
	}
	if err := redisCache.Blacklist(t.Context(), token, time.Minute); err != nil {
		t.Fatalf("Blacklist() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", rec.Code)
	}
}
