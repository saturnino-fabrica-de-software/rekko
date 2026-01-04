package handler

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
)

// WidgetService interface for the service layer
type WidgetService interface {
	CreateSession(ctx context.Context, publicKey, origin string) (*domain.WidgetSession, error)
	ValidateSession(ctx context.Context, sessionID uuid.UUID) (*domain.WidgetSession, error)
	Verify(ctx context.Context, sessionID uuid.UUID, externalID string, imageBytes []byte) (*domain.Verification, error)
	Register(ctx context.Context, sessionID uuid.UUID, externalID string, imageBytes []byte) (*domain.Face, error)
}

// WidgetHandler handles widget-related requests
type WidgetHandler struct {
	service        WidgetService
	usageTracker   UsageTracker
	webhookService WebhookService
	logger         *slog.Logger
}

// NewWidgetHandler creates a new WidgetHandler instance
func NewWidgetHandler(
	service WidgetService,
	usageTracker UsageTracker,
	webhookService WebhookService,
	logger *slog.Logger,
) *WidgetHandler {
	return &WidgetHandler{
		service:        service,
		usageTracker:   usageTracker,
		webhookService: webhookService,
		logger:         logger,
	}
}

// CreateSessionRequest request for creating a widget session
type CreateSessionRequest struct {
	PublicKey string `json:"public_key"`
	Origin    string `json:"origin"`
}

// CreateSessionResponse response for session creation
type CreateSessionResponse struct {
	SessionID string `json:"session_id"`
	ExpiresAt string `json:"expires_at"`
}

// WidgetVerifyRequest request for widget verification
type WidgetVerifyRequest struct {
	SessionID  string `json:"session_id"`
	ExternalID string `json:"external_id"`
}

// WidgetRegisterRequest request for widget registration
type WidgetRegisterRequest struct {
	SessionID  string `json:"session_id"`
	ExternalID string `json:"external_id"`
}

// trackUsage increments usage counter asynchronously (best-effort)
func (h *WidgetHandler) trackUsage(tenantID uuid.UUID, field string) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := h.usageTracker.IncrementDaily(ctx, tenantID, time.Now().UTC(), field, 1); err != nil {
			h.logger.Warn("failed to track usage",
				"error", err,
				"tenant_id", tenantID,
				"field", field,
			)
		}
	}()
}

// dispatchWidgetEvent dispatches a widget event to webhooks (best-effort, async)
func (h *WidgetHandler) dispatchWidgetEvent(tenantID uuid.UUID, eventType string, data interface{}) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := h.webhookService.Dispatch(ctx, tenantID, eventType, data); err != nil {
			h.logger.Warn("failed to dispatch webhook event",
				"error", err,
				"tenant_id", tenantID,
				"event_type", eventType,
			)
		}
	}()
}

// CreateSession POST /v1/widget/session - create a new widget session
// @Summary Create widget session
// @Description Creates a temporary session for widget authentication
// @Tags widget
// @Accept json
// @Produce json
// @Param request body CreateSessionRequest true "Session creation request"
// @Success 201 {object} CreateSessionResponse
// @Failure 400 {object} domain.AppError
// @Failure 401 {object} domain.AppError
// @Failure 403 {object} domain.AppError
// @Router /v1/widget/session [post]
func (h *WidgetHandler) CreateSession(c *fiber.Ctx) error {
	var req CreateSessionRequest

	// 1. Parse JSON body
	if err := c.BodyParser(&req); err != nil {
		return domain.ErrValidationFailed.WithError(fmt.Errorf("invalid request body: %w", err))
	}

	// 2. Validate required fields
	if req.PublicKey == "" {
		return domain.ErrValidationFailed.WithError(errors.New("public_key is required"))
	}

	if req.Origin == "" {
		return domain.ErrValidationFailed.WithError(errors.New("origin is required"))
	}

	// 3. Call service to create session
	session, err := h.service.CreateSession(c.Context(), req.PublicKey, req.Origin)
	if err != nil {
		return err
	}

	// 4. Track usage (async)
	h.trackUsage(session.TenantID, "widget_sessions")

	// 5. Dispatch webhook event (async)
	h.dispatchWidgetEvent(session.TenantID, "widget.session_created", map[string]interface{}{
		"session_id": session.ID.String(),
		"origin":     session.Origin,
	})

	// 6. Return response
	return c.Status(fiber.StatusCreated).JSON(CreateSessionResponse{
		SessionID: session.ID.String(),
		ExpiresAt: session.ExpiresAt.Format(time.RFC3339),
	})
}

// Verify POST /v1/widget/verify - verify face using widget session
// @Summary Widget face verification
// @Description Verifies a face using a widget session (1:1 verification)
// @Tags widget
// @Accept multipart/form-data
// @Produce json
// @Param session_id formData string true "Widget session ID"
// @Param external_id formData string true "External user ID"
// @Param image formData file true "Face image"
// @Success 200 {object} VerifyResponse
// @Failure 400 {object} domain.AppError
// @Failure 401 {object} domain.AppError
// @Failure 404 {object} domain.AppError
// @Router /v1/widget/verify [post]
func (h *WidgetHandler) Verify(c *fiber.Ctx) error {
	start := time.Now()

	// 1. Extract session_id from form
	sessionIDStr := strings.TrimSpace(c.FormValue("session_id"))
	if sessionIDStr == "" {
		return domain.ErrValidationFailed.WithError(errors.New("session_id is required"))
	}

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return domain.ErrValidationFailed.WithError(fmt.Errorf("invalid session_id format: %w", err))
	}

	// 2. Extract external_id from form
	externalID := strings.TrimSpace(c.FormValue("external_id"))
	if externalID == "" {
		return domain.ErrValidationFailed.WithError(errors.New("external_id is required"))
	}

	// 3. Extract and validate image
	imageBytes, err := extractAndValidateImage(c)
	if err != nil {
		return fmt.Errorf("widget verify: %w", err)
	}

	// 4. Call service to verify
	verification, err := h.service.Verify(c.Context(), sessionID, externalID, imageBytes)
	if err != nil {
		return err
	}

	// 5. Get session for tenant_id (for tracking)
	session, _ := h.service.ValidateSession(c.Context(), sessionID)
	if session != nil {
		// 6. Track usage (async)
		h.trackUsage(session.TenantID, "widget_verifications")

		// 7. Dispatch webhook event (async)
		elapsed := time.Since(start)
		h.dispatchWidgetEvent(session.TenantID, "widget.verified", map[string]interface{}{
			"verified":    verification.Verified,
			"confidence":  verification.Confidence,
			"external_id": externalID,
			"session_id":  sessionID.String(),
			"latency_ms":  elapsed.Milliseconds(),
		})
	}

	// 8. Return response
	return c.JSON(VerifyResponse{
		Verified:       verification.Verified,
		Confidence:     verification.Confidence,
		VerificationID: verification.ID.String(),
		LatencyMs:      verification.LatencyMs,
	})
}

// Register POST /v1/widget/register - register face using widget session
// @Summary Widget face registration
// @Description Registers a new face using a widget session
// @Tags widget
// @Accept multipart/form-data
// @Produce json
// @Param session_id formData string true "Widget session ID"
// @Param external_id formData string true "External user ID"
// @Param image formData file true "Face image"
// @Success 201 {object} RegisterResponse
// @Failure 400 {object} domain.AppError
// @Failure 401 {object} domain.AppError
// @Failure 404 {object} domain.AppError
// @Router /v1/widget/register [post]
func (h *WidgetHandler) Register(c *fiber.Ctx) error {
	// 1. Extract session_id from form
	sessionIDStr := strings.TrimSpace(c.FormValue("session_id"))
	if sessionIDStr == "" {
		return domain.ErrValidationFailed.WithError(errors.New("session_id is required"))
	}

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return domain.ErrValidationFailed.WithError(fmt.Errorf("invalid session_id format: %w", err))
	}

	// 2. Extract external_id from form
	externalID := strings.TrimSpace(c.FormValue("external_id"))
	if externalID == "" {
		return domain.ErrValidationFailed.WithError(errors.New("external_id is required"))
	}

	// 3. Extract and validate image
	imageBytes, err := extractAndValidateImage(c)
	if err != nil {
		return fmt.Errorf("widget register: %w", err)
	}

	// 4. Call service to register
	face, err := h.service.Register(c.Context(), sessionID, externalID, imageBytes)
	if err != nil {
		return err
	}

	// 5. Get session for tenant_id (for tracking)
	session, _ := h.service.ValidateSession(c.Context(), sessionID)
	if session != nil {
		// 6. Track usage (async)
		h.trackUsage(session.TenantID, "widget_registrations")

		// 7. Dispatch webhook event (async)
		h.dispatchWidgetEvent(session.TenantID, "widget.registered", map[string]interface{}{
			"face_id":       face.ID.String(),
			"external_id":   face.ExternalID,
			"quality_score": face.QualityScore,
			"session_id":    sessionID.String(),
		})
	}

	// 8. Return response
	return c.Status(fiber.StatusCreated).JSON(RegisterResponse{
		FaceID:       face.ID.String(),
		ExternalID:   face.ExternalID,
		QualityScore: face.QualityScore,
		CreatedAt:    face.CreatedAt.Format("2006-01-02T15:04:05Z"),
	})
}
