package router

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"pilot-backend/internal/config"
	"pilot-backend/pkg/cache"
	"pilot-backend/pkg/db"
	"pilot-backend/pkg/jwt"
	"pilot-backend/pkg/logger"
)

func newTestRouter(t *testing.T) *Router {
	t.Helper()

	cfg := config.Load()
	log := logger.New("error")

	gormDB, err := db.Connect(cfg, log)
	if err != nil {
		t.Skipf("mysql not reachable, skipping router test: %v", err)
	}
	sqlDB, _ := gormDB.DB()
	t.Cleanup(func() { sqlDB.Close() })

	redisCache := cache.NewRedis(cfg)
	if err := redisCache.Ping(t.Context()); err != nil {
		t.Skipf("redis not reachable, skipping router test: %v", err)
	}
	t.Cleanup(func() { redisCache.Close() })

	jwtMgr := jwt.NewManager("test-secret")

	return New(cfg, log, jwtMgr, redisCache, gormDB)
}

func TestRouter_Health(t *testing.T) {
	r := newTestRouter(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	r.Engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body = %s", rec.Code, rec.Body.String())
	}

	var body map[string]string
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if body["status"] != "healthy" || body["database"] != "connected" || body["cache"] != "connected" {
		t.Errorf("health body = %+v, want all connected/healthy", body)
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID header missing (RequestLogger not wired?)")
	}
}

// TestRouter_ChainAppliesToProtectedRoutes registers a throwaway route under
// r.Auth to prove the full middleware chain (logging, CORS, auth) composes
// correctly for handlers that opt into authentication — the pattern T8/T9/T10
// will use for their real routes.
func TestRouter_ChainAppliesToProtectedRoutes(t *testing.T) {
	r := newTestRouter(t)
	r.API.GET("/whoami", r.Auth, func(c *gin.Context) {})

	req := httptest.NewRequest(http.MethodGet, "/api/whoami", nil)
	rec := httptest.NewRecorder()
	r.Engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (no token)", rec.Code)
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("X-Request-ID header missing on protected route")
	}
}
