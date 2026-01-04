package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
	"github.com/saturnino-fabrica-de-software/rekko/internal/provider"
)

type FaceRepositoryInterface interface {
	Create(ctx context.Context, face *domain.Face) error
	GetByExternalID(ctx context.Context, tenantID uuid.UUID, externalID string) (*domain.Face, error)
	Delete(ctx context.Context, tenantID uuid.UUID, externalID string) error
	SearchByEmbedding(ctx context.Context, tenantID uuid.UUID, embedding []float64, threshold float64, limit int) ([]domain.SearchMatch, error)
	CountByTenant(ctx context.Context, tenantID uuid.UUID) (int, error)
}

type VerificationRepositoryInterface interface {
	Create(ctx context.Context, v *domain.Verification) error
}

type SearchAuditRepositoryInterface interface {
	Create(ctx context.Context, audit *domain.SearchAudit) error
}

type FaceService struct {
	faceRepo         FaceRepositoryInterface
	verificationRepo VerificationRepositoryInterface
	searchAuditRepo  SearchAuditRepositoryInterface
	provider         provider.FaceProvider
	threshold        float64
}

func NewFaceService(
	faceRepo FaceRepositoryInterface,
	verificationRepo VerificationRepositoryInterface,
	searchAuditRepo SearchAuditRepositoryInterface,
	faceProvider provider.FaceProvider,
) *FaceService {
	return &FaceService{
		faceRepo:         faceRepo,
		verificationRepo: verificationRepo,
		searchAuditRepo:  searchAuditRepo,
		provider:         faceProvider,
		threshold:        0.8,
	}
}

func (s *FaceService) WithThreshold(threshold float64) *FaceService {
	s.threshold = threshold
	return s
}

func (s *FaceService) Register(ctx context.Context, tenantID uuid.UUID, externalID string, imageBytes []byte, requireLiveness bool, livenessThreshold float64) (*domain.Face, error) {
	detectedFaces, err := s.provider.DetectFaces(ctx, imageBytes)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: detect faces: %w", tenantID, err)
	}

	if len(detectedFaces) == 0 {
		return nil, domain.ErrNoFaceDetected
	}

	if len(detectedFaces) > 1 {
		return nil, domain.ErrMultipleFaces
	}

	// Check liveness if required
	if requireLiveness {
		livenessResult, err := s.provider.CheckLiveness(ctx, imageBytes, livenessThreshold)
		if err != nil {
			return nil, fmt.Errorf("tenant %s: check liveness: %w", tenantID, err)
		}

		if !livenessResult.IsLive || livenessResult.Confidence < livenessThreshold {
			return nil, domain.ErrLivenessFailed
		}
	}

	_, embedding, err := s.provider.IndexFace(ctx, imageBytes)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: index face: %w", tenantID, err)
	}

	face := &domain.Face{
		TenantID:     tenantID,
		ExternalID:   externalID,
		Embedding:    embedding,
		QualityScore: detectedFaces[0].QualityScore,
	}

	if err := s.faceRepo.Create(ctx, face); err != nil {
		return nil, err
	}

	return face, nil
}

func (s *FaceService) Verify(ctx context.Context, tenantID uuid.UUID, externalID string, imageBytes []byte) (*domain.Verification, error) {
	start := time.Now()

	storedFace, err := s.faceRepo.GetByExternalID(ctx, tenantID, externalID)
	if err != nil {
		return nil, err
	}

	detectedFaces, err := s.provider.DetectFaces(ctx, imageBytes)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: detect faces: %w", tenantID, err)
	}

	if len(detectedFaces) == 0 {
		return nil, domain.ErrNoFaceDetected
	}

	if len(detectedFaces) > 1 {
		return nil, domain.ErrMultipleFaces
	}

	_, newEmbedding, err := s.provider.IndexFace(ctx, imageBytes)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: index face for verification: %w", tenantID, err)
	}

	similarity, err := s.provider.CompareFaces(ctx, storedFace.Embedding, newEmbedding)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: compare faces: %w", tenantID, err)
	}

	verified := similarity >= s.threshold
	latencyMs := time.Since(start).Milliseconds()

	verification := &domain.Verification{
		TenantID:   tenantID,
		FaceID:     &storedFace.ID,
		ExternalID: externalID,
		Verified:   verified,
		Confidence: similarity,
		LatencyMs:  latencyMs,
	}

	// Audit log - error is intentionally not returned
	// The verification result was already determined successfully
	// In production, this would be logged with proper observability
	_ = s.verificationRepo.Create(ctx, verification)

	return verification, nil
}

func (s *FaceService) Delete(ctx context.Context, tenantID uuid.UUID, externalID string) error {
	// Verify face exists and belongs to tenant before deleting
	if _, err := s.faceRepo.GetByExternalID(ctx, tenantID, externalID); err != nil {
		return err
	}

	// Delete from database
	if err := s.faceRepo.Delete(ctx, tenantID, externalID); err != nil {
		return fmt.Errorf("tenant %s: delete face: %w", tenantID, err)
	}

	return nil
}

func (s *FaceService) CheckLiveness(ctx context.Context, imageBytes []byte, threshold float64) (*domain.LivenessResult, error) {
	// Call provider to check liveness
	providerResult, err := s.provider.CheckLiveness(ctx, imageBytes, threshold)
	if err != nil {
		return nil, fmt.Errorf("check liveness: %w", err)
	}

	// Convert provider result to domain result
	result := &domain.LivenessResult{
		IsLive:     providerResult.IsLive,
		Confidence: providerResult.Confidence,
		Reasons:    providerResult.Reasons,
		Checks: domain.LivenessChecks{
			EyesOpen:     providerResult.Checks.EyesOpen,
			FacingCamera: providerResult.Checks.FacingCamera,
			QualityOK:    providerResult.Checks.QualityOK,
			SingleFace:   providerResult.Checks.SingleFace,
		},
	}

	return result, nil
}

// Search performs a 1:N face search against all faces in the tenant
// Returns matches above threshold, ordered by similarity
func (s *FaceService) Search(ctx context.Context, tenant *domain.Tenant, imageBytes []byte, threshold float64, maxResults int, clientIP string) (*domain.SearchResult, error) {
	start := time.Now()

	// 1. Extract settings from tenant
	settings := extractTenantSettings(tenant)

	// 2. Verify if search is enabled
	if !settings.SearchEnabled {
		return nil, domain.ErrSearchNotEnabled
	}

	// 3. Apply defaults if not provided
	if threshold <= 0 {
		threshold = settings.SearchThreshold
	}
	if maxResults <= 0 {
		maxResults = settings.SearchMaxResults
	}

	// 4. Validate parameters
	if threshold < 0 || threshold > 1 {
		return nil, domain.ErrInvalidThreshold
	}
	if maxResults < 1 || maxResults > 50 {
		return nil, domain.ErrInvalidMaxResults
	}

	// 5. Optional: check liveness if configured
	if settings.SearchRequireLiveness {
		liveness, err := s.provider.CheckLiveness(ctx, imageBytes, settings.LivenessThreshold)
		if err != nil {
			return nil, fmt.Errorf("tenant %s: check liveness: %w", tenant.ID, err)
		}
		if !liveness.IsLive {
			return nil, domain.ErrLivenessFailed
		}
	}

	// 6. Extract embedding from image
	_, embedding, err := s.provider.IndexFace(ctx, imageBytes)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: index face for search: %w", tenant.ID, err)
	}

	// 7. Search similar faces in database
	matches, err := s.faceRepo.SearchByEmbedding(ctx, tenant.ID, embedding, threshold, maxResults)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: search faces: %w", tenant.ID, err)
	}

	// 8. Count total faces in tenant
	totalFaces, err := s.faceRepo.CountByTenant(ctx, tenant.ID)
	if err != nil {
		return nil, fmt.Errorf("tenant %s: count faces: %w", tenant.ID, err)
	}

	// 9. Calculate latency
	latencyMs := time.Since(start).Milliseconds()
	searchID := uuid.New()

	// 10. Create audit log (async, best-effort with panic recovery)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				slog.Error("panic in search audit", "panic", r, "tenant_id", tenant.ID, "search_id", searchID)
			}
		}()
		s.createSearchAudit(tenant.ID, searchID, matches, threshold, maxResults, latencyMs, clientIP)
	}()

	// 11. Return result
	return &domain.SearchResult{
		Matches:    matches,
		TotalFaces: totalFaces,
		LatencyMs:  latencyMs,
		SearchID:   searchID,
	}, nil
}

// createSearchAudit creates an audit log entry asynchronously
func (s *FaceService) createSearchAudit(tenantID, searchID uuid.UUID, matches []domain.SearchMatch, threshold float64, maxResults int, latencyMs int64, clientIP string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	audit := &domain.SearchAudit{
		ID:           searchID,
		TenantID:     tenantID,
		ResultsCount: len(matches),
		Threshold:    threshold,
		MaxResults:   maxResults,
		LatencyMs:    latencyMs,
		ClientIP:     clientIP,
	}

	// Add top match if exists
	if len(matches) > 0 {
		audit.TopMatchExternalID = &matches[0].ExternalID
		audit.TopMatchSimilarity = &matches[0].Similarity
	}

	// Best-effort audit log creation (errors are intentionally ignored)
	_ = s.searchAuditRepo.Create(ctx, audit)
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

	// Extract search_enabled
	if val, ok := tenant.Settings["search_enabled"].(bool); ok {
		settings.SearchEnabled = val
	}

	// Extract search_require_liveness
	if val, ok := tenant.Settings["search_require_liveness"].(bool); ok {
		settings.SearchRequireLiveness = val
	}

	// Extract search_threshold
	if val, ok := tenant.Settings["search_threshold"].(float64); ok {
		settings.SearchThreshold = val
	}

	// Extract search_max_results
	if val, ok := tenant.Settings["search_max_results"].(float64); ok {
		settings.SearchMaxResults = int(val)
	}

	// Extract search_rate_limit
	if val, ok := tenant.Settings["search_rate_limit"].(float64); ok {
		settings.SearchRateLimit = int(val)
	}

	return settings
}
