package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"pilot-backend/pkg/logger"
)

// RequestLogger assigns a request_id to every request (propagated via the
// X-Request-ID response header and the gin context, for error responses and
// downstream logging to correlate), then logs method/path/status/duration
// in JSON once the request completes. It never logs headers, query strings,
// or bodies, which may carry tokens or other sensitive data.
func RequestLogger(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.NewString()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		start := time.Now()
		c.Next()
		duration := time.Since(start)

		log.Infow("http.request",
			"request_id", requestID,
			"method", c.Request.Method,
			"path", c.Request.URL.Path,
			"status", c.Writer.Status(),
			"duration_ms", duration.Milliseconds(),
			"client_ip", c.ClientIP(),
		)
	}
}
