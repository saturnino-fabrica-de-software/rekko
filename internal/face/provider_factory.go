package face

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/config"
	"github.com/saturnino-fabrica-de-software/rekko/internal/provider"
	"github.com/saturnino-fabrica-de-software/rekko/internal/provider/deepface"
	"github.com/saturnino-fabrica-de-software/rekko/internal/provider/rekognition"
)

// ProviderType defines supported face recognition provider types
type ProviderType string

const (
	// ProviderTypeDeepFace is the DeepFace provider (local, for dev/test)
	ProviderTypeDeepFace ProviderType = "deepface"
	// ProviderTypeRekognition is the AWS Rekognition provider (cloud, for prod)
	ProviderTypeRekognition ProviderType = "rekognition"
)

// NewFaceProvider creates a FaceProvider instance based on configuration
// For Rekognition, tenantID is required to create tenant-specific collections
// For DeepFace, tenantID is ignored as it's a stateless provider
//
// Environment variables:
//   - FACE_PROVIDER: "deepface" or "rekognition" (default: "deepface")
//   - DEEPFACE_URL: DeepFace API URL (default: "http://localhost:5000")
//   - AWS_REGION: AWS region for Rekognition (default: "us-east-1")
//   - AWS_ACCESS_KEY_ID: AWS credentials (via AWS SDK credential chain)
//   - AWS_SECRET_ACCESS_KEY: AWS credentials (via AWS SDK credential chain)
func NewFaceProvider(ctx context.Context, cfg *config.Config, tenantID uuid.UUID) (provider.FaceProvider, error) {
	providerType := ProviderType(cfg.FaceProvider)

	switch providerType {
	case ProviderTypeRekognition:
		return createRekognitionProvider(ctx, cfg, tenantID)

	case ProviderTypeDeepFace, "":
		// Default to DeepFace for dev/test environments
		return createDeepFaceProvider(cfg), nil

	default:
		return nil, fmt.Errorf("unknown provider type: %s (supported: %s, %s)",
			cfg.FaceProvider, ProviderTypeDeepFace, ProviderTypeRekognition)
	}
}

// createRekognitionProvider creates an AWS Rekognition provider instance
func createRekognitionProvider(ctx context.Context, cfg *config.Config, tenantID uuid.UUID) (provider.FaceProvider, error) {
	rekogConfig := rekognition.Config{
		Region:           cfg.AWSRegion,
		CollectionPrefix: "rekko-",
	}

	prov, err := rekognition.NewProvider(ctx, rekogConfig, tenantID)
	if err != nil {
		return nil, fmt.Errorf("create rekognition provider for tenant %s: %w", tenantID, err)
	}

	return prov, nil
}

// createDeepFaceProvider creates a DeepFace provider instance
func createDeepFaceProvider(cfg *config.Config) provider.FaceProvider {
	deepfaceConfig := deepface.Config{
		BaseURL: cfg.DeepFaceURL,
	}

	// Use defaults for other fields (timeout, model, detector, retry)
	if deepfaceConfig.BaseURL == "" {
		deepfaceConfig.BaseURL = deepface.DefaultConfig().BaseURL
	}

	return deepface.NewProvider(deepfaceConfig)
}
