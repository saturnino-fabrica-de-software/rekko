package handler

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/api/middleware"
	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

const (
	maxImageSize = 10 * 1024 * 1024 // 10MB
)

var validImageTypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// FaceService interface for the service
type FaceService interface {
	Register(ctx context.Context, tenantID uuid.UUID, externalID string, imageBytes []byte, requireLiveness bool, livenessThreshold float64) (*domain.Face, error)
	Verify(ctx context.Context, tenantID uuid.UUID, externalID string, imageBytes []byte) (*domain.Verification, error)
	Delete(ctx context.Context, tenantID uuid.UUID, externalID string) error
	CheckLiveness(ctx context.Context, imageBytes []byte, threshold float64) (*domain.LivenessResult, error)
}

// FaceHandler handles face-related requests
type FaceHandler struct {
	service FaceService
}

// NewFaceHandler creates a new FaceHandler instance
func NewFaceHandler(service FaceService) *FaceHandler {
	return &FaceHandler{service: service}
}

// RegisterResponse response for register endpoint
type RegisterResponse struct {
	FaceID       string  `json:"face_id"`
	ExternalID   string  `json:"external_id"`
	QualityScore float64 `json:"quality_score"`
	CreatedAt    string  `json:"created_at"`
}

// VerifyResponse response for verify endpoint
type VerifyResponse struct {
	Verified       bool    `json:"verified"`
	Confidence     float64 `json:"confidence"`
	VerificationID string  `json:"verification_id"`
	LatencyMs      int64   `json:"latency_ms"`
}

// LivenessResponse response for liveness check endpoint
type LivenessResponse struct {
	IsLive     bool                   `json:"is_live"`
	Confidence float64                `json:"confidence"`
	Checks     LivenessChecksResponse `json:"checks"`
	Reasons    []string               `json:"reasons,omitempty"`
}

// LivenessChecksResponse represents individual liveness checks in response
type LivenessChecksResponse struct {
	EyesOpen     bool `json:"eyes_open"`
	FacingCamera bool `json:"facing_camera"`
	QualityOK    bool `json:"quality_ok"`
	SingleFace   bool `json:"single_face"`
}

// Register POST /v1/faces - register a new face
func (h *FaceHandler) Register(c *fiber.Ctx) error {
	// 1. Extract tenant from context (already authenticated by middleware)
	tenant, err := middleware.GetTenant(c)
	if err != nil {
		return err
	}

	// 2. Extract external_id from form
	externalID := strings.TrimSpace(c.FormValue("external_id"))
	if externalID == "" {
		return domain.ErrValidationFailed.WithError(errors.New("external_id is required"))
	}

	// 3. Extract and validate image
	imageBytes, err := extractAndValidateImage(c)
	if err != nil {
		return fmt.Errorf("register face: %w", err)
	}

	// 4. Extract liveness settings from tenant
	settings := extractTenantSettings(tenant)

	// 5. Call service to register
	face, err := h.service.Register(c.Context(), tenant.ID, externalID, imageBytes, settings.RequireLiveness, settings.LivenessThreshold)
	if err != nil {
		return err
	}

	// 6. Return response
	return c.Status(fiber.StatusCreated).JSON(RegisterResponse{
		FaceID:       face.ID.String(),
		ExternalID:   face.ExternalID,
		QualityScore: face.QualityScore,
		CreatedAt:    face.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}

// Verify POST /v1/faces/verify - verify face 1:1
func (h *FaceHandler) Verify(c *fiber.Ctx) error {
	// 1. Extract tenant_id from context
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}

	// 2. Extract external_id from form
	externalID := strings.TrimSpace(c.FormValue("external_id"))
	if externalID == "" {
		return domain.ErrValidationFailed.WithError(errors.New("external_id is required"))
	}

	// 3. Extract and validate image
	imageBytes, err := extractAndValidateImage(c)
	if err != nil {
		return fmt.Errorf("verify face: %w", err)
	}

	// 4. Call service to verify
	verification, err := h.service.Verify(c.Context(), tenantID, externalID, imageBytes)
	if err != nil {
		return err
	}

	// 5. Return response
	return c.JSON(VerifyResponse{
		Verified:       verification.Verified,
		Confidence:     verification.Confidence,
		VerificationID: verification.ID.String(),
		LatencyMs:      verification.LatencyMs,
	})
}

// Delete DELETE /v1/faces/:external_id - delete face (LGPD)
func (h *FaceHandler) Delete(c *fiber.Ctx) error {
	// 1. Extract tenant_id from context
	tenantID, err := middleware.GetTenantID(c)
	if err != nil {
		return err
	}

	// 2. Extract external_id from URL
	externalID := strings.TrimSpace(c.Params("external_id"))
	if externalID == "" {
		return domain.ErrValidationFailed.WithError(errors.New("external_id is required"))
	}

	// 3. Call service to delete
	if err := h.service.Delete(c.Context(), tenantID, externalID); err != nil {
		return err
	}

	// 4. Return 204 No Content
	return c.SendStatus(fiber.StatusNoContent)
}

// CheckLiveness POST /v1/faces/liveness - check if image contains a live person
func (h *FaceHandler) CheckLiveness(c *fiber.Ctx) error {
	// 1. Extract tenant from context
	tenant, err := middleware.GetTenant(c)
	if err != nil {
		return err
	}

	// 2. Extract and validate image
	imageBytes, err := extractAndValidateImage(c)
	if err != nil {
		return fmt.Errorf("check liveness: %w", err)
	}

	// 3. Get liveness threshold from tenant settings
	settings := extractTenantSettings(tenant)

	// 4. Call provider to check liveness
	result, err := h.service.CheckLiveness(c.Context(), imageBytes, settings.LivenessThreshold)
	if err != nil {
		return err
	}

	// 5. Return response
	return c.JSON(LivenessResponse{
		IsLive:     result.IsLive,
		Confidence: result.Confidence,
		Checks: LivenessChecksResponse{
			EyesOpen:     result.Checks.EyesOpen,
			FacingCamera: result.Checks.FacingCamera,
			QualityOK:    result.Checks.QualityOK,
			SingleFace:   result.Checks.SingleFace,
		},
		Reasons: result.Reasons,
	})
}

// extractAndValidateImage extracts and validates the image from the form
func extractAndValidateImage(c *fiber.Ctx) ([]byte, error) {
	// 1. Extract file
	file, err := c.FormFile("image")
	if err != nil {
		return nil, domain.ErrValidationFailed.WithError(err)
	}

	// 2. Validate size
	if file.Size > maxImageSize {
		return nil, domain.ErrInvalidImage.WithError(nil)
	}

	if file.Size == 0 {
		return nil, domain.ErrInvalidImage.WithError(nil)
	}

	// 3. Validate Content-Type
	contentType := file.Header.Get("Content-Type")
	if !validImageTypes[contentType] {
		return nil, domain.ErrInvalidImage.WithError(nil)
	}

	// 4. Read image bytes
	f, err := file.Open()
	if err != nil {
		return nil, domain.ErrInvalidImage.WithError(err)
	}
	defer func() {
		_ = f.Close()
	}()

	imageBytes, err := io.ReadAll(f)
	if err != nil {
		return nil, domain.ErrInvalidImage.WithError(err)
	}

	return imageBytes, nil
}

// extractTenantSettings extracts TenantSettings from tenant Settings map
func extractTenantSettings(tenant *domain.Tenant) domain.TenantSettings {
	settings := domain.DefaultTenantSettings()

	if tenant.Settings == nil {
		return settings
	}

	// Extract verification_threshold
	if val, ok := tenant.Settings["verification_threshold"].(float64); ok {
		settings.VerificationThreshold = val
	}

	// Extract max_faces_per_user
	if val, ok := tenant.Settings["max_faces_per_user"].(float64); ok {
		settings.MaxFacesPerUser = int(val)
	}

	// Extract require_liveness
	if val, ok := tenant.Settings["require_liveness"].(bool); ok {
		settings.RequireLiveness = val
	}

	// Extract liveness_threshold
	if val, ok := tenant.Settings["liveness_threshold"].(float64); ok {
		settings.LivenessThreshold = val
	}

	return settings
}
