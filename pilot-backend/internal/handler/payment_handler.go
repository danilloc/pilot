package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"pilot-backend/internal/models"
	"pilot-backend/internal/service"
)

// PaymentHandler serves /api/payments/*, all of which require authentication.
type PaymentHandler struct {
	paymentService *service.PaymentService
}

// NewPaymentHandler builds a PaymentHandler from its collaborators.
func NewPaymentHandler(paymentService *service.PaymentService) *PaymentHandler {
	return &PaymentHandler{paymentService: paymentService}
}

// Register mounts the payment routes on api, all behind auth.
func (h *PaymentHandler) Register(api *gin.RouterGroup, auth gin.HandlerFunc) {
	group := api.Group("/payments")
	group.Use(auth)
	group.GET("", h.List)
	group.POST("/sync", h.Sync)
}

// List handles GET /api/payments.
func (h *PaymentHandler) List(c *gin.Context) {
	start, end, err := parseDateRange(c.Query("start_date"), c.Query("end_date"))
	if err != nil {
		c.Error(models.NewAPIError(models.ErrCodeInvalidQueryParam, "invalid start_date/end_date (expected YYYY-MM-DD)"))
		return
	}
	limit, offset := parsePagination(c.Query("limit"), c.Query("offset"))

	payments, pagination, err := h.paymentService.ListPayments(c.Request.Context(), c.GetInt64("driver_id"), start, end, limit, offset)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": payments, "pagination": pagination})
}

type syncPaymentsRequest struct {
	Limit int `json:"limit"`
}

// Sync handles POST /api/payments/sync.
func (h *PaymentHandler) Sync(c *gin.Context) {
	var req syncPaymentsRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.Error(models.NewAPIError(models.ErrCodeInvalidQueryParam, "invalid request body"))
			return
		}
	}

	count, syncedAt, err := h.paymentService.SyncPayments(c.Request.Context(), c.GetInt64("driver_id"), req.Limit)
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
