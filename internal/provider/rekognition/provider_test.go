package rekognition

import (
	"context"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/saturnino-fabrica-de-software/rekko/internal/provider"
)

// TestProviderImplementsInterface verifies that Provider implements FaceProvider
func TestProviderImplementsInterface(t *testing.T) {
	var _ provider.FaceProvider = (*Provider)(nil)
}

// TestDefaultConfig verifies default configuration values
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "us-east-1", cfg.Region)
	assert.Equal(t, "rekko-", cfg.CollectionPrefix)
}

// TestCollectionName verifies collection name generation
func TestCollectionName(t *testing.T) {
	tests := []struct {
		name     string
		config   Config
		tenantID string
		want     string
	}{
		{
			name:     "default prefix",
			config:   DefaultConfig(),
			tenantID: "550e8400-e29b-41d4-a716-446655440000",
			want:     "rekko-550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name: "custom prefix",
			config: Config{
				Region:           "us-west-2",
				CollectionPrefix: "custom-",
			},
			tenantID: "tenant-123",
			want:     "custom-tenant-123",
		},
		{
			name: "empty prefix",
			config: Config{
				Region:           "eu-west-1",
				CollectionPrefix: "",
			},
			tenantID: "test-tenant",
			want:     "test-tenant",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.config.CollectionName(tt.tenantID)
			assert.Equal(t, tt.want, got)
		})
	}
}

// TestErrors verifies error definitions
func TestErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		msg  string
	}{
		{
			name: "CollectionNotFound",
			err:  ErrCollectionNotFound,
			msg:  "collection not found",
		},
		{
			name: "CollectionAlreadyExists",
			err:  ErrCollectionAlreadyExists,
			msg:  "already exists",
		},
		{
			name: "InvalidCredentials",
			err:  ErrInvalidCredentials,
			msg:  "invalid",
		},
		{
			name: "NoFaceDetected",
			err:  ErrNoFaceDetected,
			msg:  "no face detected",
		},
		{
			name: "MultipleFaces",
			err:  ErrMultipleFaces,
			msg:  "multiple faces",
		},
		{
			name: "FaceNotFound",
			err:  ErrFaceNotFound,
			msg:  "not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Contains(t, tt.err.Error(), tt.msg)
		})
	}
}

// TestParseIndexFacesError verifies error parsing from unindexed faces
func TestParseIndexFacesError(t *testing.T) {
	tests := []struct {
		name            string
		unindexedFaces  []types.UnindexedFace
		wantErr         error
		wantErrContains string
	}{
		{
			name:            "no unindexed faces",
			unindexedFaces:  []types.UnindexedFace{},
			wantErr:         nil,
			wantErrContains: "",
		},
		{
			name: "exceeds max faces",
			unindexedFaces: []types.UnindexedFace{
				{
					Reasons: []types.Reason{types.ReasonExceedsMaxFaces},
				},
			},
			wantErr:         ErrMultipleFaces,
			wantErrContains: "",
		},
		{
			name: "extreme pose",
			unindexedFaces: []types.UnindexedFace{
				{
					Reasons: []types.Reason{types.ReasonExtremePose},
				},
			},
			wantErr:         ErrNoFaceDetected,
			wantErrContains: "EXTREME_POSE",
		},
		{
			name: "low brightness",
			unindexedFaces: []types.UnindexedFace{
				{
					Reasons: []types.Reason{types.ReasonLowBrightness},
				},
			},
			wantErr:         ErrNoFaceDetected,
			wantErrContains: "LOW_BRIGHTNESS",
		},
		{
			name: "low sharpness",
			unindexedFaces: []types.UnindexedFace{
				{
					Reasons: []types.Reason{types.ReasonLowSharpness},
				},
			},
			wantErr:         ErrNoFaceDetected,
			wantErrContains: "LOW_SHARPNESS",
		},
		{
			name: "low confidence",
			unindexedFaces: []types.UnindexedFace{
				{
					Reasons: []types.Reason{types.ReasonLowConfidence},
				},
			},
			wantErr:         ErrNoFaceDetected,
			wantErrContains: "LOW_CONFIDENCE",
		},
		{
			name: "small bounding box",
			unindexedFaces: []types.UnindexedFace{
				{
					Reasons: []types.Reason{types.ReasonSmallBoundingBox},
				},
			},
			wantErr:         ErrNoFaceDetected,
			wantErrContains: "SMALL_BOUNDING_BOX",
		},
		{
			name: "low face quality",
			unindexedFaces: []types.UnindexedFace{
				{
					Reasons: []types.Reason{types.ReasonLowFaceQuality},
				},
			},
			wantErr:         ErrNoFaceDetected,
			wantErrContains: "LOW_FACE_QUALITY",
		},
		{
			name: "no reasons",
			unindexedFaces: []types.UnindexedFace{
				{
					Reasons: []types.Reason{},
				},
			},
			wantErr:         ErrNoFaceDetected,
			wantErrContains: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParseIndexFacesError(tt.unindexedFaces)

			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)

			if tt.wantErrContains != "" {
				assert.Contains(t, err.Error(), tt.wantErrContains)
			}
		})
	}
}

// TestCalculateQualityScore verifies quality score calculation
func TestCalculateQualityScore(t *testing.T) {
	p := &Provider{}

	tests := []struct {
		name       string
		quality    *types.ImageQuality
		want       float64
		wantMin    float64
		wantMax    float64
		exactMatch bool
	}{
		{
			name:       "nil quality",
			quality:    nil,
			want:       0.0,
			exactMatch: true,
		},
		{
			name: "perfect quality",
			quality: &types.ImageQuality{
				Brightness: ptr(float32(100.0)),
				Sharpness:  ptr(float32(100.0)),
			},
			want:       1.0,
			exactMatch: true,
		},
		{
			name: "zero quality",
			quality: &types.ImageQuality{
				Brightness: ptr(float32(0.0)),
				Sharpness:  ptr(float32(0.0)),
			},
			want:       0.0,
			exactMatch: true,
		},
		{
			name: "medium quality",
			quality: &types.ImageQuality{
				Brightness: ptr(float32(50.0)),
				Sharpness:  ptr(float32(50.0)),
			},
			want:       0.5,
			exactMatch: true,
		},
		{
			name: "sharpness weighted more heavily",
			quality: &types.ImageQuality{
				Brightness: ptr(float32(100.0)), // 30% weight
				Sharpness:  ptr(float32(0.0)),   // 70% weight
			},
			want:       0.3,
			exactMatch: true,
		},
		{
			name: "brightness low, sharpness high",
			quality: &types.ImageQuality{
				Brightness: ptr(float32(0.0)),
				Sharpness:  ptr(float32(100.0)),
			},
			want:       0.7,
			exactMatch: true,
		},
		{
			name: "only brightness set",
			quality: &types.ImageQuality{
				Brightness: ptr(float32(80.0)),
			},
			wantMin: 0.23,
			wantMax: 0.25,
		},
		{
			name: "only sharpness set",
			quality: &types.ImageQuality{
				Sharpness: ptr(float32(80.0)),
			},
			wantMin: 0.55,
			wantMax: 0.57,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := p.calculateQualityScore(tt.quality)

			if tt.exactMatch {
				assert.InDelta(t, tt.want, got, 0.0001)
			} else {
				assert.GreaterOrEqual(t, got, tt.wantMin)
				assert.LessOrEqual(t, got, tt.wantMax)
			}
		})
	}
}

// TestCompareFaces_NotSupported verifies that CompareFaces with embeddings returns error
func TestCompareFaces_NotSupported(t *testing.T) {
	tenantID := uuid.New()
	p := &Provider{
		tenantID: tenantID,
	}

	similarity, err := p.CompareFaces(context.Background(), []float64{1.0}, []float64{1.0})

	assert.Error(t, err)
	assert.Equal(t, 0.0, similarity)
	assert.Contains(t, err.Error(), "not supported")
	assert.Contains(t, err.Error(), tenantID.String())
}

// TestParseNoFaceError verifies error parsing for no face detected scenarios
func TestParseNoFaceError(t *testing.T) {
	tests := []struct {
		name           string
		err            error
		wantErr        error
		wantErrMessage string
	}{
		{
			name:    "nil error",
			err:     nil,
			wantErr: nil,
		},
		{
			name:    "non-AWS error",
			err:     assert.AnError,
			wantErr: assert.AnError,
		},
		// Note: Testing with real AWS errors would require creating smithy.APIError instances
		// which is complex. The actual behavior is tested in integration tests.
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ParseNoFaceError(tt.err)

			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}

			require.Error(t, err)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// skipIfNoAWSCredentials skips the test if AWS credentials are not configured
func skipIfNoAWSCredentials(t *testing.T) {
	t.Helper()

	if os.Getenv("AWS_ACCESS_KEY_ID") == "" {
		t.Skip("Skipping integration test: AWS_ACCESS_KEY_ID not set")
	}
}

// TestIntegration_CreateDeleteCollection tests collection lifecycle
func TestIntegration_CreateDeleteCollection(t *testing.T) {
	skipIfNoAWSCredentials(t)

	ctx := context.Background()
	tenantID := uuid.New().String() // Unique tenant for this test

	cfg := DefaultConfig()
	client, err := NewClient(ctx, cfg)
	require.NoError(t, err)

	// Ensure clean state
	_ = client.DeleteCollection(ctx, tenantID)

	// Create collection
	err = client.CreateCollection(ctx, tenantID)
	require.NoError(t, err)

	// Verify collection exists
	exists, err := client.CollectionExists(ctx, tenantID)
	require.NoError(t, err)
	assert.True(t, exists)

	// Verify face count is zero
	count, err := client.GetCollectionFaceCount(ctx, tenantID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Delete collection
	err = client.DeleteCollection(ctx, tenantID)
	require.NoError(t, err)

	// Verify collection is deleted
	exists, err = client.CollectionExists(ctx, tenantID)
	require.NoError(t, err)
	assert.False(t, exists)
}

// TestIntegration_EnsureCollection tests idempotent collection creation
func TestIntegration_EnsureCollection(t *testing.T) {
	skipIfNoAWSCredentials(t)

	ctx := context.Background()
	tenantID := uuid.New().String()

	cfg := DefaultConfig()
	client, err := NewClient(ctx, cfg)
	require.NoError(t, err)

	// Clean up any existing collection
	_ = client.DeleteCollection(ctx, tenantID)

	// First ensure should create
	err = client.EnsureCollection(ctx, tenantID)
	require.NoError(t, err)

	exists, err := client.CollectionExists(ctx, tenantID)
	require.NoError(t, err)
	assert.True(t, exists)

	// Second ensure should be no-op
	err = client.EnsureCollection(ctx, tenantID)
	require.NoError(t, err)

	// Clean up
	_ = client.DeleteCollection(ctx, tenantID)
}

// TestIntegration_ListCollections tests listing collections
func TestIntegration_ListCollections(t *testing.T) {
	skipIfNoAWSCredentials(t)

	ctx := context.Background()
	tenantID := uuid.New().String()

	cfg := DefaultConfig()
	client, err := NewClient(ctx, cfg)
	require.NoError(t, err)

	// Create a test collection
	err = client.CreateCollection(ctx, tenantID)
	require.NoError(t, err)
	defer func() { _ = client.DeleteCollection(ctx, tenantID) }() // Clean up

	// List collections
	collections, err := client.ListCollections(ctx)
	require.NoError(t, err)

	// Should include our test collection
	expectedName := cfg.CollectionName(tenantID)
	found := false
	for _, name := range collections {
		if name == expectedName {
			found = true
			break
		}
	}
	assert.True(t, found, "expected to find collection %s in list", expectedName)
}

// TestIntegration_CollectionNotFound tests operations on non-existent collection
func TestIntegration_CollectionNotFound(t *testing.T) {
	skipIfNoAWSCredentials(t)

	ctx := context.Background()
	nonExistentTenant := uuid.New().String()

	cfg := DefaultConfig()
	client, err := NewClient(ctx, cfg)
	require.NoError(t, err)

	// Ensure collection doesn't exist
	_ = client.DeleteCollection(ctx, nonExistentTenant)

	// Delete non-existent collection
	err = client.DeleteCollection(ctx, nonExistentTenant)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrCollectionNotFound)

	// Get face count from non-existent collection
	_, err = client.GetCollectionFaceCount(ctx, nonExistentTenant)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrCollectionNotFound)
}

// TestIntegration_CollectionAlreadyExists tests duplicate collection creation
func TestIntegration_CollectionAlreadyExists(t *testing.T) {
	skipIfNoAWSCredentials(t)

	ctx := context.Background()
	tenantID := uuid.New().String()

	cfg := DefaultConfig()
	client, err := NewClient(ctx, cfg)
	require.NoError(t, err)

	// Clean up
	_ = client.DeleteCollection(ctx, tenantID)

	// Create collection
	err = client.CreateCollection(ctx, tenantID)
	require.NoError(t, err)
	defer func() { _ = client.DeleteCollection(ctx, tenantID) }()

	// Try to create again - should fail
	err = client.CreateCollection(ctx, tenantID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, ErrCollectionAlreadyExists)
}

// TestIntegration_NewProvider tests provider initialization
func TestIntegration_NewProvider(t *testing.T) {
	skipIfNoAWSCredentials(t)

	ctx := context.Background()
	tenantID := uuid.New()

	cfg := DefaultConfig()
	p, err := NewProvider(ctx, cfg, tenantID)
	require.NoError(t, err)
	require.NotNil(t, p)

	// Verify provider fields
	assert.Equal(t, tenantID, p.tenantID)
	assert.NotNil(t, p.client)

	// Verify collection was created
	exists, err := p.client.CollectionExists(ctx, tenantID.String())
	require.NoError(t, err)
	assert.True(t, exists)

	// Get face count
	count, err := p.GetFaceCount(ctx)
	require.NoError(t, err)
	assert.Equal(t, int64(0), count)

	// Clean up
	_ = p.DeleteCollection(ctx)
}

// ptr is a helper function to get pointer to a value
func ptr[T any](v T) *T {
	return &v
}
