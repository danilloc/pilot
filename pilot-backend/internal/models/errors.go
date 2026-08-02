package models

import (
	"fmt"
	"net/http"
	"strings"
)

// ErrorCode is a stable, machine-readable identifier for an API error.
type ErrorCode string

const (
	// Auth errors
	ErrCodeInvalidToken     ErrorCode = "AUTH_001"
	ErrCodeTokenExpired     ErrorCode = "AUTH_002"
	ErrCodeTokenBlacklisted ErrorCode = "AUTH_003"
	ErrCodeOAuthFailed      ErrorCode = "AUTH_004"
	ErrCodeDriverBanned     ErrorCode = "AUTH_005"

	// Validation errors
	ErrCodeInvalidEmail        ErrorCode = "VAL_001"
	ErrCodeInvalidPhone        ErrorCode = "VAL_002"
	ErrCodeInvalidRating       ErrorCode = "VAL_003"
	ErrCodeInvalidDistance     ErrorCode = "VAL_004"
	ErrCodeInvalidFare         ErrorCode = "VAL_005"
	ErrCodeInvalidTimeRange    ErrorCode = "VAL_006"
	ErrCodeInvalidGoalAmount   ErrorCode = "VAL_007"
	ErrCodeInvalidPaymentValue ErrorCode = "VAL_008"

	// Database errors
	ErrCodeDatabaseError ErrorCode = "DB_001"
	ErrCodeNotFound      ErrorCode = "DB_002"

	// Server errors
	ErrCodeInternalError ErrorCode = "ERR_500"
	ErrCodeEncryption    ErrorCode = "ERR_ENC"
)

// Sentinel validation errors returned by model Validate() methods. Each
// wraps an ErrorCode so handlers can propagate it straight to an API
// response without re-mapping error strings.
var (
	ErrInvalidEmail        = NewAPIError(ErrCodeInvalidEmail, "invalid email")
	ErrInvalidPhone        = NewAPIError(ErrCodeInvalidPhone, "invalid phone number (expected E.164)")
	ErrInvalidRating       = NewAPIError(ErrCodeInvalidRating, "rating must be between 0 and 5")
	ErrDriverBanned        = NewAPIError(ErrCodeDriverBanned, "account suspended")
	ErrInvalidDistance     = NewAPIError(ErrCodeInvalidDistance, "distance must be greater than zero")
	ErrInvalidFare         = NewAPIError(ErrCodeInvalidFare, "fare value cannot be negative")
	ErrInvalidTimeRange    = NewAPIError(ErrCodeInvalidTimeRange, "trip end time must not precede start time")
	ErrInvalidGoalAmount   = NewAPIError(ErrCodeInvalidGoalAmount, "goal amount must be greater than zero")
	ErrInvalidPaymentValue = NewAPIError(ErrCodeInvalidPaymentValue, "payment amount, tolls and taxes cannot be negative")
)

// APIError is the shape returned to clients on failure. It never carries a
// stack trace or internal error detail — those belong in server-side logs only.
type APIError struct {
	Code      ErrorCode `json:"code"`
	Message   string    `json:"message"`
	RequestID string    `json:"request_id,omitempty"`
	Timestamp string    `json:"timestamp,omitempty"`
}

// NewAPIError builds an APIError with the given code and message.
func NewAPIError(code ErrorCode, message string) *APIError {
	return &APIError{
		Code:    code,
		Message: message,
	}
}

func (e *APIError) Error() string {
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// StatusCode maps an ErrorCode onto the HTTP status the API should respond
// with, by the code's namespace prefix (AUTH_ -> 401, VAL_ -> 400, ...).
// Unrecognized codes default to 500, the safe choice for an unmapped
// server-side failure.
func (c ErrorCode) StatusCode() int {
	switch {
	case c == ErrCodeNotFound:
		return http.StatusNotFound
	case strings.HasPrefix(string(c), "AUTH_"):
		return http.StatusUnauthorized
	case strings.HasPrefix(string(c), "VAL_"):
		return http.StatusBadRequest
	case c == "RATE_001":
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}
