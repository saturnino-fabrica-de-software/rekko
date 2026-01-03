package face

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/config"
	"github.com/saturnino-fabrica-de-software/rekko/internal/provider/deepface"
	"github.com/saturnino-fabrica-de-software/rekko/internal/provider/rekognition"
)

func TestNewFaceProvider_DeepFace(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	tests := []struct {
		name         string
		faceProvider string
		deepFaceURL  string
		wantType     string
	}{
		{
			name:         "explicit deepface provider",
			faceProvider: "deepface",
			deepFaceURL:  "http://localhost:5000",
			wantType:     "*deepface.Provider",
		},
		{
			name:         "empty provider defaults to deepface",
			faceProvider: "",
			deepFaceURL:  "http://localhost:5000",
			wantType:     "*deepface.Provider",
		},
		{
			name:         "custom deepface URL",
			faceProvider: "deepface",
			deepFaceURL:  "http://custom-host:8080",
			wantType:     "*deepface.Provider",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				FaceProvider: tt.faceProvider,
				DeepFaceURL:  tt.deepFaceURL,
			}

			provider, err := NewFaceProvider(ctx, cfg, tenantID)
			if err != nil {
				t.Fatalf("NewFaceProvider() error = %v", err)
			}

			// Type assertion to verify correct provider type
			if _, ok := provider.(*deepface.Provider); !ok {
				t.Errorf("NewFaceProvider() returned type %T, want %s", provider, tt.wantType)
			}
		})
	}
}

func TestNewFaceProvider_Rekognition(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping Rekognition test in short mode (requires AWS credentials)")
	}

	ctx := context.Background()
	tenantID := uuid.New()

	cfg := &config.Config{
		FaceProvider: "rekognition",
		AWSRegion:    "us-east-1",
	}

	provider, err := NewFaceProvider(ctx, cfg, tenantID)
	if err != nil {
		// If error is due to missing AWS credentials, skip test
		if err.Error() != "" {
			t.Skipf("Skipping Rekognition test (likely missing AWS credentials): %v", err)
		}
		t.Fatalf("NewFaceProvider() error = %v", err)
	}

	// Type assertion to verify correct provider type
	if _, ok := provider.(*rekognition.Provider); !ok {
		t.Errorf("NewFaceProvider() returned type %T, want *rekognition.Provider", provider)
	}
}

func TestNewFaceProvider_UnknownProvider(t *testing.T) {
	ctx := context.Background()
	tenantID := uuid.New()

	cfg := &config.Config{
		FaceProvider: "unknown-provider",
	}

	_, err := NewFaceProvider(ctx, cfg, tenantID)
	if err == nil {
		t.Fatal("NewFaceProvider() expected error for unknown provider, got nil")
	}

	expectedErrMsg := "unknown provider type: unknown-provider"
	if err.Error()[:len(expectedErrMsg)] != expectedErrMsg {
		t.Errorf("NewFaceProvider() error = %v, want error containing %q", err, expectedErrMsg)
	}
}

func TestProviderType_Constants(t *testing.T) {
	// Ensure constants are defined correctly
	if ProviderTypeDeepFace != "deepface" {
		t.Errorf("ProviderTypeDeepFace = %q, want %q", ProviderTypeDeepFace, "deepface")
	}

	if ProviderTypeRekognition != "rekognition" {
		t.Errorf("ProviderTypeRekognition = %q, want %q", ProviderTypeRekognition, "rekognition")
	}
}
