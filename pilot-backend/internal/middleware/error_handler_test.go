package middleware

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"pilot-backend/internal/models"
	"pilot-backend/pkg/logger"
)

func TestErrorHandler_GenericErrorHidesDetail(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(ErrorHandler(logger.New("error", "test")))
	engine.GET("/boom", func(c *gin.Context) {
		c.Error(errors.New("leaked db password: hunter2"))
	})

	req := httptest.NewRequest(http.MethodGet, "/boom", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", rec.Code)
	}

	var apiErr models.APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if apiErr.Message == "leaked db password: hunter2" {
		t.Error("ErrorHandler leaked the internal error message to the client")
	}
	if apiErr.Code != models.ErrCodeInternalError {
		t.Errorf("Code = %q, want %q", apiErr.Code, models.ErrCodeInternalError)
	}
}

func TestErrorHandler_PropagatesAPIError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(ErrorHandler(logger.New("error", "test")))
	engine.GET("/not-found", func(c *gin.Context) {
		c.Error(models.NewAPIError(models.ErrCodeNotFound, "driver not found"))
	})

	req := httptest.NewRequest(http.MethodGet, "/not-found", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", rec.Code)
	}

	var apiErr models.APIError
	if err := json.Unmarshal(rec.Body.Bytes(), &apiErr); err != nil {
		t.Fatalf("Unmarshal() error = %v", err)
	}
	if apiErr.Code != models.ErrCodeNotFound {
		t.Errorf("Code = %q, want %q", apiErr.Code, models.ErrCodeNotFound)
	}
}

func TestErrorHandler_NoErrorPassesThrough(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.Use(ErrorHandler(logger.New("error", "test")))
	engine.GET("/ok", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true}) })

	req := httptest.NewRequest(http.MethodGet, "/ok", nil)
	rec := httptest.NewRecorder()
	engine.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", rec.Code)
	}
}
