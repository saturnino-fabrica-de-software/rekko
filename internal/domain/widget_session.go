package domain

import (
	"time"

	"github.com/google/uuid"
)

// WidgetSession represents a temporary session for widget authentication
type WidgetSession struct {
	ID        uuid.UUID `json:"id"`
	TenantID  uuid.UUID `json:"tenant_id"`
	Origin    string    `json:"origin"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// IsExpired checks if the session has expired
func (s *WidgetSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// Widget-specific errors
var (
	ErrWidgetSessionNotFound = &AppError{
		Code:       "WIDGET_SESSION_NOT_FOUND",
		Message:    "Widget session not found or expired",
		StatusCode: 404,
	}

	ErrWidgetSessionExpired = &AppError{
		Code:       "WIDGET_SESSION_EXPIRED",
		Message:    "Widget session has expired",
		StatusCode: 401,
	}

	ErrInvalidPublicKey = &AppError{
		Code:       "INVALID_PUBLIC_KEY",
		Message:    "Invalid or inactive public key",
		StatusCode: 401,
	}

	ErrOriginNotAllowed = &AppError{
		Code:       "ORIGIN_NOT_ALLOWED",
		Message:    "Origin domain is not allowed for this tenant",
		StatusCode: 403,
	}

	ErrInvalidOrigin = &AppError{
		Code:       "INVALID_ORIGIN",
		Message:    "Invalid origin format",
		StatusCode: 422,
	}
)
