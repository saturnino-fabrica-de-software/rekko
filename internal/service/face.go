package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/domain"
	"github.com/saturnino-fabrica-de-software/rekko/internal/provider"
)

type FaceRepositoryInterface interface {
	Create(ctx context.Context, face *domain.Face) error
	GetByExternalID(ctx context.Context, tenantID uuid.UUID, externalID string) (*domain.Face, error)
	Delete(ctx context.Context, tenantID uuid.UUID, externalID string) error
}

type VerificationRepositoryInterface interface {
	Create(ctx context.Context, v *domain.Verification) error
}

type FaceService struct {
	faceRepo         FaceRepositoryInterface
	verificationRepo VerificationRepositoryInterface
	provider         provider.FaceProvider
	threshold        float64
}

func NewFaceService(
	faceRepo FaceRepositoryInterface,
	verificationRepo VerificationRepositoryInterface,
	faceProvider provider.FaceProvider,
) *FaceService {
	return &FaceService{
		faceRepo:         faceRepo,
		verificationRepo: verificationRepo,
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
