package router

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

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

// TestRouter_RateLimitersMatchDesignDoc proves the per-category rate
// limiters router.New() builds carry the exact limits from
// api-endpoints/design.md's Rate Limiting table (Auth 5/min, Trips 30/min,
// Stats 60/min, Goals 30/min), not just that the rate-limit primitive works
// in isolation (already covered by internal/middleware's tests). Each
// limiter reports its configured ceiling via the X-RateLimit-Limit
// response header, so one request per category is enough evidence — no
// need to fire dozens of requests to find each boundary.
func TestRouter_RateLimitersMatchDesignDoc(t *testing.T) {
	r := newTestRouter(t)

	tests := []struct {
		name      string
		limiter   gin.HandlerFunc
		wantLimit string
	}{
		{"auth", r.RateLimit.Auth, "5"},
		{"trips", r.RateLimit.Trips, "30"},
		{"stats", r.RateLimit.Stats, "60"},
		{"goals", r.RateLimit.Goals, "30"},
	}

	nonce := time.Now().UnixNano()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := fmt.Sprintf("/rl-%s-%d", tt.name, nonce)
			r.API.GET(path, tt.limiter, func(c *gin.Context) { c.Status(http.StatusOK) })

			req := httptest.NewRequest(http.MethodGet, "/api"+path, nil)
			rec := httptest.NewRecorder()
			r.Engine.ServeHTTP(rec, req)

			if rec.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200, body = %s", rec.Code, rec.Body.String())
			}
			if got := rec.Header().Get("X-RateLimit-Limit"); got != tt.wantLimit {
				t.Errorf("X-RateLimit-Limit = %q, want %q", got, tt.wantLimit)
			}
		})
	}
}

// TestRouter_AuthRateLimit_BlocksAfterFive proves the Auth limiter actually
// enforces design.md's 5 req/min ceiling end-to-end through the router
// (not just that the header reports "5").
func TestRouter_AuthRateLimit_BlocksAfterFive(t *testing.T) {
	r := newTestRouter(t)
	path := fmt.Sprintf("/rl-auth-enforce-%d", time.Now().UnixNano())
	r.API.GET(path, r.RateLimit.Auth, func(c *gin.Context) { c.Status(http.StatusOK) })

	var lastCode int
	for i := 0; i < 6; i++ {
		req := httptest.NewRequest(http.MethodGet, "/api"+path, nil)
		rec := httptest.NewRecorder()
		r.Engine.ServeHTTP(rec, req)
		lastCode = rec.Code
		if i < 5 && rec.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i+1, rec.Code)
		}
	}
	if lastCode != http.StatusTooManyRequests {
		t.Fatalf("6th request status = %d, want 429", lastCode)
	}
}
