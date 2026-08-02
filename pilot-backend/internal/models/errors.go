package models

import "fmt"

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
	ErrCodeInvalidEmail    ErrorCode = "VAL_001"
	ErrCodeInvalidPhone    ErrorCode = "VAL_002"
	ErrCodeInvalidRating   ErrorCode = "VAL_003"
	ErrCodeInvalidDistance ErrorCode = "VAL_004"
	ErrCodeInvalidFare     ErrorCode = "VAL_005"

	// Database errors
	ErrCodeDatabaseError ErrorCode = "DB_001"
	ErrCodeNotFound      ErrorCode = "DB_002"

	// Server errors
	ErrCodeInternalError ErrorCode = "ERR_500"
	ErrCodeEncryption    ErrorCode = "ERR_ENC"
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
