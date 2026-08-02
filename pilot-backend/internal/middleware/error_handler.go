package middleware

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"pilot-backend/internal/models"
	"pilot-backend/pkg/logger"
)

// ErrorHandler catches any error a handler attached via c.Error(err) that
// wasn't already turned into a response, logs the real error server-side,
// and returns a generic APIError to the client — full detail (including
// stack-trace-worthy context) never crosses the wire.
func ErrorHandler(log *logger.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if len(c.Errors) == 0 || c.Writer.Written() {
			return
		}

		err := c.Errors.Last().Err
		log.Errorw("request_error", "error", err.Error(), "path", c.Request.URL.Path, "request_id", c.GetString("request_id"))

		var apiErr *models.APIError
		if !errors.As(err, &apiErr) {
			apiErr = models.NewAPIError(models.ErrCodeInternalError, "something went wrong")
		}
		apiErr.RequestID = c.GetString("request_id")
		apiErr.Timestamp = time.Now().UTC().Format(time.RFC3339)

		status := http.StatusInternalServerError
		if apiErr.Code == models.ErrCodeNotFound {
			status = http.StatusNotFound
		}

		c.JSON(status, apiErr)
	}
}
