package middleware

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"pilot-backend/internal/models"
	"pilot-backend/pkg/cache"
	"pilot-backend/pkg/jwt"
)

const authContextTimeout = 5 * time.Second

// JWTAuth validates the Bearer token on every request: signature, expiry,
// and blacklist status. On success it stores driver_id and client_ip in the
// gin context for downstream handlers/middleware (e.g. RateLimit).
func JWTAuth(jwtMgr *jwt.Manager, redis *cache.Redis) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			respondUnauthorized(c, models.ErrCodeInvalidToken, "missing token")
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" || parts[1] == "" {
			respondUnauthorized(c, models.ErrCodeInvalidToken, "invalid token format")
			return
		}
		token := parts[1]

		ctx, cancel := context.WithTimeout(c.Request.Context(), authContextTimeout)
		defer cancel()

		blacklisted, err := redis.IsBlacklisted(ctx, token)
		if err != nil {
			respondError(c, 500, models.ErrCodeInternalError, "internal server error")
			return
		}
		if blacklisted {
			respondUnauthorized(c, models.ErrCodeTokenBlacklisted, "token no longer valid")
			return
		}

		claims, err := jwtMgr.ValidateToken(token)
		if err != nil {
			if errors.Is(err, jwt.ErrTokenExpired) {
				respondUnauthorized(c, models.ErrCodeTokenExpired, "token expired")
				return
			}
			respondUnauthorized(c, models.ErrCodeInvalidToken, "invalid token")
			return
		}

		c.Set("driver_id", claims.DriverID)
		c.Set("client_ip", c.ClientIP())
		c.Set("jwt_token", token)
		c.Next()
	}
}

func respondUnauthorized(c *gin.Context, code models.ErrorCode, message string) {
	respondError(c, 401, code, message)
}

func respondError(c *gin.Context, status int, code models.ErrorCode, message string) {
	apiErr := models.NewAPIError(code, message)
	apiErr.RequestID = c.GetString("request_id")
	apiErr.Timestamp = time.Now().UTC().Format(time.RFC3339)
	c.AbortWithStatusJSON(status, apiErr)
}
