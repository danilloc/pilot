package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"pilot-backend/internal/models"
	"pilot-backend/internal/service"
)

const (
	dateLayout          = "2006-01-02"
	defaultListLimit    = 50
	maxListLimit        = 100
	defaultLookbackDays = 30
)

// TripHandler serves /api/trips/*, all of which require authentication.
type TripHandler struct {
	tripService *service.TripService
}

// NewTripHandler builds a TripHandler from its collaborators.
func NewTripHandler(tripService *service.TripService) *TripHandler {
	return &TripHandler{tripService: tripService}
}

// Register mounts the trip routes on api, all behind auth. rateLimit
// should be the "Trips" category limiter (30 req/min per design.md).
func (h *TripHandler) Register(api *gin.RouterGroup, auth, rateLimit gin.HandlerFunc) {
	group := api.Group("/trips")
	group.Use(auth, rateLimit)
	group.GET("", h.List)
	group.GET("/:id", h.Get)
	group.POST("/sync", h.Sync)
}

// List handles GET /api/trips.
func (h *TripHandler) List(c *gin.Context) {
	start, end, err := parseDateRange(c.Query("start_date"), c.Query("end_date"))
	if err != nil {
		c.Error(models.NewAPIError(models.ErrCodeInvalidQueryParam, "invalid start_date/end_date (expected YYYY-MM-DD)"))
		return
	}
	limit, offset := parsePagination(c.Query("limit"), c.Query("offset"))
	status := c.Query("status")

	trips, pagination, err := h.tripService.ListTrips(c.Request.Context(), c.GetInt64("driver_id"), start, end, status, limit, offset)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": trips, "pagination": pagination})
}

// Get handles GET /api/trips/:id.
func (h *TripHandler) Get(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(models.NewAPIError(models.ErrCodeNotFound, "trip not found"))
		return
	}

	trip, err := h.tripService.GetTrip(c.Request.Context(), c.GetInt64("driver_id"), id)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, trip)
}

type syncTripsRequest struct {
	Limit int `json:"limit"`
}

// Sync handles POST /api/trips/sync.
func (h *TripHandler) Sync(c *gin.Context) {
	var req syncTripsRequest
	// Body is optional (defaults to defaultSyncLimit inside the service);
	// a malformed body is still an error, an absent one is not.
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Error(models.NewAPIError(models.ErrCodeInvalidQueryParam, "invalid request body"))
			return
		}
	}

	count, syncedAt, err := h.tripService.SyncTrips(c.Request.Context(), c.GetInt64("driver_id"), req.Limit)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"synced_count": count,
		"synced_at":    syncedAt.UTC().Format(time.RFC3339),
	})
}

// parseDateRange parses start/end as YYYY-MM-DD, defaulting to the last
// defaultLookbackDays when omitted. end is inclusive through 23:59:59.
func parseDateRange(startStr, endStr string) (time.Time, time.Time, error) {
	end := time.Now()
	if endStr != "" {
		parsed, err := time.Parse(dateLayout, endStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		end = parsed
	}
	end = time.Date(end.Year(), end.Month(), end.Day(), 23, 59, 59, 0, end.Location())

	start := end.AddDate(0, 0, -defaultLookbackDays)
	if startStr != "" {
		parsed, err := time.Parse(dateLayout, startStr)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		start = parsed
	}

	return start, end, nil
}

// parsePagination parses limit/offset query params, clamping limit to
// [1, maxListLimit] and offset to >= 0.
func parsePagination(limitStr, offsetStr string) (int, int) {
	limit := defaultListLimit
	if v, err := strconv.Atoi(limitStr); err == nil && v > 0 {
		limit = v
	}
	if limit > maxListLimit {
		limit = maxListLimit
	}

	offset := 0
	if v, err := strconv.Atoi(offsetStr); err == nil && v >= 0 {
		offset = v
	}

	return limit, offset
}
