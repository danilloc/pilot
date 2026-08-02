package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"pilot-backend/internal/service"
)

// StatsHandler serves /api/stats/*, all of which require authentication.
type StatsHandler struct {
	statsService *service.StatsService
}

// NewStatsHandler builds a StatsHandler from its collaborators.
func NewStatsHandler(statsService *service.StatsService) *StatsHandler {
	return &StatsHandler{statsService: statsService}
}

// Register mounts the stats routes on api, all behind auth. rateLimit
// should be the "Stats" category limiter (60 req/min per design.md).
func (h *StatsHandler) Register(api *gin.RouterGroup, auth, rateLimit gin.HandlerFunc) {
	group := api.Group("/stats")
	group.Use(auth, rateLimit)
	group.GET("/today", h.Today)
	group.GET("/week", h.Week)
	group.GET("/month", h.Month)
}

// Today handles GET /api/stats/today.
func (h *StatsHandler) Today(c *gin.Context) {
	stats, err := h.statsService.Today(c.Request.Context(), c.GetInt64("driver_id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, stats)
}

// Week handles GET /api/stats/week.
func (h *StatsHandler) Week(c *gin.Context) {
	stats, err := h.statsService.Week(c.Request.Context(), c.GetInt64("driver_id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, stats)
}

// Month handles GET /api/stats/month. Optional ?month=YYYY-MM, defaults to
// the current month.
func (h *StatsHandler) Month(c *gin.Context) {
	stats, err := h.statsService.Month(c.Request.Context(), c.GetInt64("driver_id"), c.Query("month"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, stats)
}
