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
	Register(ctx context.Context, sessionID uuid.UUID, externalID string, imageBytes []byte) (*domain.Face, error)
	ValidateLiveness(ctx context.Context, sessionID uuid.UUID, imageBytes []byte) (*domain.LivenessResult, error)
	Search(ctx context.Context, sessionID uuid.UUID, imageBytes []byte, clientIP string) (*domain.SearchResult, error)
	CheckRegistration(ctx context.Context, sessionID uuid.UUID, externalID string) (*domain.RegistrationCheck, error)
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

// WidgetRegisterRequest request for widget registration
type WidgetRegisterRequest struct {
	SessionID  string `json:"session_id"`
	ExternalID string `json:"external_id"`
}

// WidgetLivenessResponse response for widget liveness validation
type WidgetLivenessResponse struct {
	IsLive     bool                         `json:"is_live"`
	Confidence float64                      `json:"confidence"`
	Checks     WidgetLivenessChecksResponse `json:"checks"`
	Reasons    []string                     `json:"reasons,omitempty"`
}

// WidgetLivenessChecksResponse represents individual liveness checks
type WidgetLivenessChecksResponse struct {
	EyesOpen     bool `json:"eyes_open"`
	FacingCamera bool `json:"facing_camera"`
	QualityOK    bool `json:"quality_ok"`
	SingleFace   bool `json:"single_face"`
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

// ValidateLiveness POST /v1/widget/validate - validate liveness using widget session
// @Summary Widget liveness validation
// @Description Validates that the image contains a live person (anti-spoofing)
// @Tags widget
// @Accept multipart/form-data
// @Produce json
// @Param session_id formData string true "Widget session ID"
// @Param image formData file true "Face image"
// @Success 200 {object} WidgetLivenessResponse
// @Failure 400 {object} domain.AppError
// @Failure 401 {object} domain.AppError
// @Router /v1/widget/validate [post]
func (h *WidgetHandler) ValidateLiveness(c *fiber.Ctx) error {
	// 1. Extract session_id from form
	sessionIDStr := strings.TrimSpace(c.FormValue("session_id"))
	if sessionIDStr == "" {
		return domain.ErrValidationFailed.WithError(errors.New("session_id is required"))
	}

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return domain.ErrValidationFailed.WithError(fmt.Errorf("invalid session_id format: %w", err))
	}

	// 2. Extract and validate image
	imageBytes, err := extractAndValidateImage(c)
	if err != nil {
		return fmt.Errorf("widget validate liveness: %w", err)
	}

	// 3. Call service to validate liveness
	result, err := h.service.ValidateLiveness(c.Context(), sessionID, imageBytes)
	if err != nil {
		return err
	}

	// 4. Get session for tenant_id (for tracking)
	session, _ := h.service.ValidateSession(c.Context(), sessionID)
	if session != nil {
		// 5. Track usage (async)
		h.trackUsage(session.TenantID, "widget_liveness_checks")

		// 6. Dispatch webhook event (async)
		h.dispatchWidgetEvent(session.TenantID, "widget.liveness_validated", map[string]interface{}{
			"is_live":    result.IsLive,
			"confidence": result.Confidence,
			"session_id": sessionID.String(),
		})
	}

	// 7. Return response
	return c.JSON(WidgetLivenessResponse{
		IsLive:     result.IsLive,
		Confidence: result.Confidence,
		Checks: WidgetLivenessChecksResponse{
			EyesOpen:     result.Checks.EyesOpen,
			FacingCamera: result.Checks.FacingCamera,
			QualityOK:    result.Checks.QualityOK,
			SingleFace:   result.Checks.SingleFace,
		},
		Reasons: result.Reasons,
	})
}

// WidgetSearchResponse represents the response for widget search (identify) operation
type WidgetSearchResponse struct {
	Identified bool    `json:"identified"`
	ExternalID string  `json:"external_id,omitempty"`
	Confidence float64 `json:"confidence,omitempty"`
}

// Search godoc
// @Summary Search for a face in the tenant's database (1:N identification)
// @Description Identifies a person by their face without requiring an external ID
// @Tags Widget
// @Accept multipart/form-data
// @Produce json
// @Param session_id formData string true "Widget session ID"
// @Param image formData file true "Face image"
// @Success 200 {object} WidgetSearchResponse
// @Failure 400 {object} domain.AppError
// @Failure 401 {object} domain.AppError
// @Failure 404 {object} domain.AppError "No matching face found"
// @Router /v1/widget/search [post]
func (h *WidgetHandler) Search(c *fiber.Ctx) error {
	// 1. Extract session_id from form
	sessionIDStr := strings.TrimSpace(c.FormValue("session_id"))
	if sessionIDStr == "" {
		return domain.ErrValidationFailed.WithError(errors.New("session_id is required"))
	}

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return domain.ErrValidationFailed.WithError(fmt.Errorf("invalid session_id format: %w", err))
	}

	// 2. Extract and validate image
	imageBytes, err := extractAndValidateImage(c)
	if err != nil {
		return fmt.Errorf("widget search: %w", err)
	}

	// 3. Get client IP for audit
	clientIP := c.IP()

	// 4. Call service to search
	result, err := h.service.Search(c.Context(), sessionID, imageBytes, clientIP)
	if err != nil {
		return err
	}

	// 5. Get session for tenant_id (for tracking)
	session, _ := h.service.ValidateSession(c.Context(), sessionID)
	if session != nil {
		// 6. Track usage (async)
		h.trackUsage(session.TenantID, "widget_searches")

		// 7. Dispatch webhook event (async)
		identified := len(result.Matches) > 0
		eventData := map[string]interface{}{
			"identified": identified,
			"session_id": sessionID.String(),
		}
		if identified {
			eventData["external_id"] = result.Matches[0].ExternalID
			eventData["confidence"] = result.Matches[0].Similarity
		}
		h.dispatchWidgetEvent(session.TenantID, "widget.searched", eventData)
	}

	// 8. Return response
	if len(result.Matches) == 0 {
		return c.JSON(WidgetSearchResponse{
			Identified: false,
		})
	}

	match := result.Matches[0]
	return c.JSON(WidgetSearchResponse{
		Identified: true,
		ExternalID: match.ExternalID,
		Confidence: match.Similarity,
	})
}

// CheckRegistrationResponse response for registration check
type CheckRegistrationResponse struct {
	Registered   bool   `json:"registered"`
	RegisteredAt string `json:"registered_at,omitempty"`
}

// CheckRegistration GET /v1/widget/check - check if external_id is registered
// @Summary Check if face is registered
// @Description Checks if a face is already registered for the given external_id
// @Tags widget
// @Produce json
// @Param session_id query string true "Widget session ID"
// @Param external_id query string true "External user ID to check"
// @Success 200 {object} CheckRegistrationResponse
// @Failure 400 {object} domain.AppError
// @Failure 401 {object} domain.AppError
// @Router /v1/widget/check [get]
func (h *WidgetHandler) CheckRegistration(c *fiber.Ctx) error {
	// 1. Extract session_id from query
	sessionIDStr := strings.TrimSpace(c.Query("session_id"))
	if sessionIDStr == "" {
		return domain.ErrValidationFailed.WithError(errors.New("session_id is required"))
	}

	sessionID, err := uuid.Parse(sessionIDStr)
	if err != nil {
		return domain.ErrValidationFailed.WithError(fmt.Errorf("invalid session_id format: %w", err))
	}

	// 2. Extract external_id from query
	externalID := strings.TrimSpace(c.Query("external_id"))
	if externalID == "" {
		return domain.ErrValidationFailed.WithError(errors.New("external_id is required"))
	}

	// 3. Call service to check registration
	result, err := h.service.CheckRegistration(c.Context(), sessionID, externalID)
	if err != nil {
		return err
	}

	// 4. Build response
	response := CheckRegistrationResponse{
		Registered: result.Registered,
	}
	if result.RegisteredAt != nil {
		response.RegisteredAt = result.RegisteredAt.Format(time.RFC3339)
	}

	return c.JSON(response)
}
