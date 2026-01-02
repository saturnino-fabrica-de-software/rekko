package domain

import (
	"fmt"
)

type AppError struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	StatusCode int    `json:"-"`
	Err        error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

func (e *AppError) WithError(err error) *AppError {
	return &AppError{
		Code:       e.Code,
		Message:    e.Message,
		StatusCode: e.StatusCode,
		Err:        err,
	}
}

// Pre-defined errors
var (
	ErrInternal = &AppError{
		Code:       "INTERNAL_ERROR",
		Message:    "An unexpected error occurred",
		StatusCode: 500,
	}

	ErrBadRequest = &AppError{
		Code:       "BAD_REQUEST",
		Message:    "Invalid request",
		StatusCode: 400,
	}

	ErrUnauthorized = &AppError{
		Code:       "UNAUTHORIZED",
		Message:    "Invalid or missing API key",
		StatusCode: 401,
	}

	ErrForbidden = &AppError{
		Code:       "FORBIDDEN",
		Message:    "Access denied",
		StatusCode: 403,
	}

	ErrNotFound = &AppError{
		Code:       "NOT_FOUND",
		Message:    "Resource not found",
		StatusCode: 404,
	}

	ErrFaceNotFound = &AppError{
		Code:       "FACE_NOT_FOUND",
		Message:    "Face not found",
		StatusCode: 404,
	}

	ErrFaceExists = &AppError{
		Code:       "FACE_ALREADY_EXISTS",
		Message:    "Face already registered for this external_id",
		StatusCode: 409,
	}

	ErrInvalidImage = &AppError{
		Code:       "INVALID_IMAGE",
		Message:    "Invalid image format or corrupted file",
		StatusCode: 422,
	}

	ErrNoFaceDetected = &AppError{
		Code:       "NO_FACE_DETECTED",
		Message:    "No face detected in the image",
		StatusCode: 422,
	}

	ErrMultipleFaces = &AppError{
		Code:       "MULTIPLE_FACES",
		Message:    "Multiple faces detected, please provide image with single face",
		StatusCode: 422,
	}

	ErrLowQualityImage = &AppError{
		Code:       "LOW_QUALITY_IMAGE",
		Message:    "Image quality too low for reliable recognition",
		StatusCode: 422,
	}

	ErrLivenessFailed = &AppError{
		Code:       "LIVENESS_FAILED",
		Message:    "Liveness check failed, possible spoofing attempt",
		StatusCode: 422,
	}

	ErrTenantNotFound = &AppError{
		Code:       "TENANT_NOT_FOUND",
		Message:    "Tenant not found",
		StatusCode: 404,
	}

	ErrRateLimitExceeded = &AppError{
		Code:       "RATE_LIMIT_EXCEEDED",
		Message:    "Rate limit exceeded, please try again later",
		StatusCode: 429,
	}

	ErrValidationFailed = &AppError{
		Code:       "VALIDATION_FAILED",
		Message:    "Request validation failed",
		StatusCode: 422,
	}
)
