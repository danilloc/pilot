package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"

	"pilot-backend/internal/models"
	"pilot-backend/pkg/cache"
)

const rateLimitContextTimeout = 3 * time.Second

// RateLimit caps requests to maxRequests per window, per client, per
// endpoint. The client key is the authenticated driver_id when JWTAuth ran
// first, falling back to the caller's IP for pre-auth endpoints (e.g.
// login). Redis INCR/EXPIRE gives an approximate fixed-window counter,
// which is sufficient for abuse protection at this scale.
func RateLimit(redis *cache.Redis, maxRequests int, window time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		client := clientKey(c)
		key := fmt.Sprintf("rate_limit:%s:%s", client, c.Request.URL.Path)

		ctx, cancel := context.WithTimeout(c.Request.Context(), rateLimitContextTimeout)
		defer cancel()

		count, err := redis.Incr(ctx, key)
		if err != nil {
			respondError(c, 500, models.ErrCodeInternalError, "internal server error")
			return
		}
		if count == 1 {
			_ = redis.Expire(ctx, key, window)
		}

		remaining := maxRequests - int(count)
		if remaining < 0 {
			remaining = 0
		}
		c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", maxRequests))
		c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))

		if count > int64(maxRequests) {
			respondError(c, 429, models.ErrorCode("RATE_001"), "too many requests")
			return
		}

		c.Next()
	}
}

func clientKey(c *gin.Context) string {
	if driverID, ok := c.Get("driver_id"); ok {
		return fmt.Sprintf("driver:%v", driverID)
	}
	return "ip:" + c.ClientIP()
}
