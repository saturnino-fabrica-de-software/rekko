package domain

import (
	"errors"
	"testing"
)

func TestAppError_Error(t *testing.T) {
	tests := []struct {
		name     string
		appErr   *AppError
		expected string
	}{
		{
			name:     "error without wrapped error",
			appErr:   ErrFaceNotFound,
			expected: "Face not found",
		},
		{
			name: "error with wrapped error",
			appErr: &AppError{
				Code:       "TEST_ERROR",
				Message:    "Test message",
				StatusCode: 500,
				Err:        errors.New("underlying error"),
			},
			expected: "Test message: underlying error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.appErr.Error(); got != tt.expected {
				t.Errorf("Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAppError_Unwrap(t *testing.T) {
	underlying := errors.New("underlying error")
	appErr := &AppError{
		Code:       "TEST",
		Message:    "test",
		StatusCode: 500,
		Err:        underlying,
	}

	if got := appErr.Unwrap(); got != underlying {
		t.Errorf("Unwrap() = %v, want %v", got, underlying)
	}

	// Test with nil error
	appErrNoWrap := ErrFaceNotFound
	if got := appErrNoWrap.Unwrap(); got != nil {
		t.Errorf("Unwrap() = %v, want nil", got)
	}
}

func TestAppError_WithError(t *testing.T) {
	underlying := errors.New("db connection failed")
	newErr := ErrInternal.WithError(underlying)

	if newErr.Code != ErrInternal.Code {
		t.Errorf("Code = %v, want %v", newErr.Code, ErrInternal.Code)
	}

	if newErr.StatusCode != ErrInternal.StatusCode {
		t.Errorf("StatusCode = %v, want %v", newErr.StatusCode, ErrInternal.StatusCode)
	}

	if newErr.Err != underlying {
		t.Errorf("Err = %v, want %v", newErr.Err, underlying)
	}

	// Check errors.Is still works
	if !errors.Is(newErr, underlying) {
		t.Errorf("errors.Is should return true for wrapped error")
	}
}

func TestErrorsIs(t *testing.T) {
	// Test that errors.As works with AppError
	err := ErrFaceNotFound.WithError(errors.New("not in db"))

	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Errorf("errors.As should match AppError")
	}

	if appErr.Code != "FACE_NOT_FOUND" {
		t.Errorf("Code = %v, want FACE_NOT_FOUND", appErr.Code)
	}
}

func TestPredefinedErrors(t *testing.T) {
	tests := []struct {
		err        *AppError
		code       string
		statusCode int
	}{
		{ErrInternal, "INTERNAL_ERROR", 500},
		{ErrBadRequest, "BAD_REQUEST", 400},
		{ErrUnauthorized, "UNAUTHORIZED", 401},
		{ErrForbidden, "FORBIDDEN", 403},
		{ErrNotFound, "NOT_FOUND", 404},
		{ErrFaceNotFound, "FACE_NOT_FOUND", 404},
		{ErrFaceExists, "FACE_ALREADY_EXISTS", 409},
		{ErrInvalidImage, "INVALID_IMAGE", 422},
		{ErrNoFaceDetected, "NO_FACE_DETECTED", 422},
		{ErrMultipleFaces, "MULTIPLE_FACES", 422},
		{ErrLowQualityImage, "LOW_QUALITY_IMAGE", 422},
		{ErrLivenessFailed, "LIVENESS_FAILED", 422},
		{ErrTenantNotFound, "TENANT_NOT_FOUND", 404},
		{ErrRateLimitExceeded, "RATE_LIMIT_EXCEEDED", 429},
		{ErrValidationFailed, "VALIDATION_FAILED", 422},
	}

	for _, tt := range tests {
		t.Run(tt.code, func(t *testing.T) {
			if tt.err.Code != tt.code {
				t.Errorf("Code = %v, want %v", tt.err.Code, tt.code)
			}
			if tt.err.StatusCode != tt.statusCode {
				t.Errorf("StatusCode = %v, want %v", tt.err.StatusCode, tt.statusCode)
			}
		})
	}
}
