package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"pilot-backend/internal/models"
	"pilot-backend/internal/service"
)

// DriverHandler serves /api/drivers/*, all of which require authentication.
type DriverHandler struct {
	driverService *service.DriverService
}

// NewDriverHandler builds a DriverHandler from its collaborators.
func NewDriverHandler(driverService *service.DriverService) *DriverHandler {
	return &DriverHandler{driverService: driverService}
}

// Register mounts the driver routes on api, all behind auth. design.md's
// Rate Limiting table doesn't call out a "Drivers" category, so rateLimit
// is expected to be the general default limiter.
func (h *DriverHandler) Register(api *gin.RouterGroup, auth, rateLimit gin.HandlerFunc) {
	group := api.Group("/drivers")
	group.Use(auth, rateLimit)
	group.GET("/me", h.Me)
	group.PUT("/me", h.UpdateMe)
	group.POST("/sync-profile", h.SyncProfile)
}

// Me handles GET /api/drivers/me.
func (h *DriverHandler) Me(c *gin.Context) {
	driver, err := h.driverService.GetProfile(c.Request.Context(), c.GetInt64("driver_id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, driver)
}

type updateDriverRequest struct {
	Phone string `json:"phone"`
}

// UpdateMe handles PUT /api/drivers/me.
func (h *DriverHandler) UpdateMe(c *gin.Context) {
	var req updateDriverRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(models.NewAPIError(models.ErrCodeInvalidPhone, "invalid request body"))
		return
	}

	driver, err := h.driverService.UpdateProfile(c.Request.Context(), c.GetInt64("driver_id"), req.Phone)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, driver)
}

// SyncProfile handles POST /api/drivers/sync-profile.
func (h *DriverHandler) SyncProfile(c *gin.Context) {
	syncedAt, err := h.driverService.SyncProfile(c.Request.Context(), c.GetInt64("driver_id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "synced_at": syncedAt.UTC().Format(time.RFC3339)})
}
