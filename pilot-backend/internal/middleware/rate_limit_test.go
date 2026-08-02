package middleware

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestRateLimit_BlocksAfterLimit(t *testing.T) {
	redisCache := testRedis(t)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	// Unique path per test run so leftover counters from previous runs
	// (Redis persists across test invocations) don't affect the assertion.
	path := fmt.Sprintf("/ping-%d", time.Now().UnixNano())
	engine.Use(RateLimit(redisCache, 3, time.Minute))
	engine.GET(path, func(c *gin.Context) { c.Status(http.StatusOK) })

	var lastCode int
	for i := 0; i < 5; i++ {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		lastCode = rec.Code
		if i < 3 && rec.Code != http.StatusOK {
			t.Fatalf("request %d: status = %d, want 200", i+1, rec.Code)
		}
	}

	if lastCode != http.StatusTooManyRequests {
		t.Fatalf("6th request status = %d, want 429", lastCode)
	}
}

func TestRateLimit_SeparatesClientsByKey(t *testing.T) {
	redisCache := testRedis(t)

	gin.SetMode(gin.TestMode)
	engine := gin.New()
	path := fmt.Sprintf("/ping2-%d", time.Now().UnixNano())
	engine.Use(func(c *gin.Context) {
		c.Set("driver_id", c.GetHeader("X-Driver-ID"))
		c.Next()
	})
	engine.Use(RateLimit(redisCache, 1, time.Minute))
	engine.GET(path, func(c *gin.Context) { c.Status(http.StatusOK) })

	for _, driver := range []string{"1", "2"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		req.Header.Set("X-Driver-ID", driver)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("driver %s: status = %d, want 200 (separate rate limit buckets)", driver, rec.Code)
		}
	}
}
