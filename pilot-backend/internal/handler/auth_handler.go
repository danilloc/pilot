// Package handler wires HTTP requests onto the service/repository layers.
package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"pilot-backend/internal/models"
	"pilot-backend/internal/repository"
	"pilot-backend/internal/service"
)

// AuthHandler serves /api/auth/*.
type AuthHandler struct {
	authService *service.AuthService
	driverRepo  *repository.DriverRepository
}

// NewAuthHandler builds an AuthHandler from its collaborators.
func NewAuthHandler(authService *service.AuthService, driverRepo *repository.DriverRepository) *AuthHandler {
	return &AuthHandler{authService: authService, driverRepo: driverRepo}
}

// Register mounts the auth routes on api. login is public; me and logout
// require auth (the caller must already have a token to log out of).
// rateLimit applies to the whole group, including the public login route —
// it runs before auth, so it's keyed by client IP throughout.
func (h *AuthHandler) Register(api *gin.RouterGroup, auth, rateLimit gin.HandlerFunc) {
	group := api.Group("/auth")
	group.Use(rateLimit)
	group.POST("/uber-login", h.UberLogin)
	group.GET("/me", auth, h.Me)
	group.POST("/logout", auth, h.Logout)
}

type uberLoginRequest struct {
	Code  string `json:"code" binding:"required"`
	State string `json:"state"`
}

// UberLogin handles POST /api/auth/uber-login.
func (h *AuthHandler) UberLogin(c *gin.Context) {
	var req uberLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(models.NewAPIError(models.ErrCodeOAuthFailed, "code is required"))
		return
	}

	token, driver, err := h.authService.LoginWithUberCode(c.Request.Context(), req.Code, c.ClientIP())
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"token": token, "driver": driver})
}

// Me handles GET /api/auth/me, returning the authenticated driver.
func (h *AuthHandler) Me(c *gin.Context) {
	driverID := c.GetInt64("driver_id")

	driver, err := h.driverRepo.GetByID(c.Request.Context(), driverID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			c.Error(models.NewAPIError(models.ErrCodeNotFound, "driver not found"))
			return
		}
		c.Error(models.NewAPIError(models.ErrCodeDatabaseError, "database error"))
		return
	}

	driver.Redact()
	c.JSON(http.StatusOK, driver)
}

// Logout handles POST /api/auth/logout, blacklisting the caller's token.
func (h *AuthHandler) Logout(c *gin.Context) {
	token := c.GetString("jwt_token")

	if err := h.authService.Logout(c.Request.Context(), token); err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
