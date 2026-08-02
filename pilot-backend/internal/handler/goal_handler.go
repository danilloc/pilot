package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"pilot-backend/internal/models"
	"pilot-backend/internal/service"
)

const goalDateLayout = "2006-01-02"

// GoalHandler serves /api/goals/*, all of which require authentication.
type GoalHandler struct {
	goalService *service.GoalService
}

// NewGoalHandler builds a GoalHandler from its collaborators.
func NewGoalHandler(goalService *service.GoalService) *GoalHandler {
	return &GoalHandler{goalService: goalService}
}

// Register mounts the goal routes on api, all behind auth.
func (h *GoalHandler) Register(api *gin.RouterGroup, auth gin.HandlerFunc) {
	group := api.Group("/goals")
	group.Use(auth)
	group.POST("", h.Create)
	group.GET("", h.Get)
	group.GET("/progress", h.Progress)
	group.PUT("/:id", h.Update)
	group.DELETE("/:id", h.Delete)
}

type createGoalRequest struct {
	GoalDate   string  `json:"goal_date" binding:"required"`
	GoalAmount float64 `json:"goal_amount" binding:"required"`
}

// Create handles POST /api/goals.
func (h *GoalHandler) Create(c *gin.Context) {
	var req createGoalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(models.NewAPIError(models.ErrCodeInvalidQueryParam, "goal_date and goal_amount are required"))
		return
	}

	goalDate, err := time.Parse(goalDateLayout, req.GoalDate)
	if err != nil {
		c.Error(models.NewAPIError(models.ErrCodeInvalidQueryParam, "invalid goal_date (expected YYYY-MM-DD)"))
		return
	}

	goal, err := h.goalService.Create(c.Request.Context(), c.GetInt64("driver_id"), goalDate, req.GoalAmount)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, goal)
}

// Get handles GET /api/goals?date=YYYY-MM-DD (defaults to today).
func (h *GoalHandler) Get(c *gin.Context) {
	date := time.Now()
	if dateStr := c.Query("date"); dateStr != "" {
		parsed, err := time.Parse(goalDateLayout, dateStr)
		if err != nil {
			c.Error(models.NewAPIError(models.ErrCodeInvalidQueryParam, "invalid date (expected YYYY-MM-DD)"))
			return
		}
		date = parsed
	}

	goal, err := h.goalService.GetByDate(c.Request.Context(), c.GetInt64("driver_id"), date)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, goal)
}

// Progress handles GET /api/goals/progress.
func (h *GoalHandler) Progress(c *gin.Context) {
	progress, err := h.goalService.Progress(c.Request.Context(), c.GetInt64("driver_id"))
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, progress)
}

type updateGoalRequest struct {
	GoalAmount float64 `json:"goal_amount" binding:"required"`
}

// Update handles PUT /api/goals/:id.
func (h *GoalHandler) Update(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(models.NewAPIError(models.ErrCodeNotFound, "goal not found"))
		return
	}

	var req updateGoalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(models.NewAPIError(models.ErrCodeInvalidQueryParam, "goal_amount is required"))
		return
	}

	goal, err := h.goalService.Update(c.Request.Context(), c.GetInt64("driver_id"), id, req.GoalAmount)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, goal)
}

// Delete handles DELETE /api/goals/:id, marking the goal ABANDONED rather
// than removing the row.
func (h *GoalHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.Error(models.NewAPIError(models.ErrCodeNotFound, "goal not found"))
		return
	}

	if err := h.goalService.Abandon(c.Request.Context(), c.GetInt64("driver_id"), id); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "status": "ABANDONED"})
}
