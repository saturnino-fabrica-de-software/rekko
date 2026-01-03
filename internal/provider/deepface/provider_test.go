package deepface

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/saturnino-fabrica-de-software/rekko/internal/provider"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProviderImplementsInterface verifies that Provider implements FaceProvider
func TestProviderImplementsInterface(t *testing.T) {
	var _ provider.FaceProvider = (*Provider)(nil)
}

// TestNewProvider verifies provider creation
func TestNewProvider(t *testing.T) {
	config := DefaultConfig()
	p := NewProvider(config)

	if p == nil {
		t.Fatal("expected provider to be created, got nil")
	}

	if p.client == nil {
		t.Fatal("expected client to be initialized, got nil")
	}
}

// TestCosineSimilarity verifies similarity calculation
func TestCosineSimilarity(t *testing.T) {
	tests := []struct {
		name       string
		embedding1 []float64
		embedding2 []float64
		want       float64
	}{
		{
			name:       "identical vectors",
			embedding1: []float64{1.0, 0.0, 0.0},
			embedding2: []float64{1.0, 0.0, 0.0},
			want:       1.0,
		},
		{
			name:       "orthogonal vectors",
			embedding1: []float64{1.0, 0.0},
			embedding2: []float64{0.0, 1.0},
			want:       0.0,
		},
		{
			name:       "opposite vectors",
			embedding1: []float64{1.0, 0.0},
			embedding2: []float64{-1.0, 0.0},
			want:       -1.0,
		},
		{
			name:       "different lengths",
			embedding1: []float64{1.0, 0.0},
			embedding2: []float64{1.0, 0.0, 0.0},
			want:       0.0,
		},
		{
			name:       "empty vectors",
			embedding1: []float64{},
			embedding2: []float64{},
			want:       0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CosineSimilarity(tt.embedding1, tt.embedding2)
			if abs(got-tt.want) > 0.0001 {
				t.Errorf("CosineSimilarity() = %v, want %v", got, tt.want)
			}
		})
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// TestProvider_DetectFaces tests face detection with mocked server
func TestProvider_DetectFaces(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse RepresentResponse
		serverStatus   int
		wantCount      int
		wantErr        bool
	}{
		{
			name: "single face detected",
			serverResponse: RepresentResponse{
				Results: []RepresentResult{
					{
						Embedding:  make([]float64, 512),
						FacialArea: FacialArea{X: 10, Y: 20, W: 200, H: 200},
					},
				},
			},
			serverStatus: http.StatusOK,
			wantCount:    1,
			wantErr:      false,
		},
		{
			name: "multiple faces detected",
			serverResponse: RepresentResponse{
				Results: []RepresentResult{
					{Embedding: make([]float64, 512), FacialArea: FacialArea{X: 10, Y: 10, W: 100, H: 100}},
					{Embedding: make([]float64, 512), FacialArea: FacialArea{X: 200, Y: 10, W: 100, H: 100}},
				},
			},
			serverStatus: http.StatusOK,
			wantCount:    2,
			wantErr:      false,
		},
		{
			name:           "no faces detected",
			serverResponse: RepresentResponse{Results: []RepresentResult{}},
			serverStatus:   http.StatusOK,
			wantCount:      0,
			wantErr:        false,
		},
		{
			name:           "server error",
			serverResponse: RepresentResponse{},
			serverStatus:   http.StatusInternalServerError,
			wantCount:      0,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			config := DefaultConfig()
			config.BaseURL = server.URL
			config.RetryCount = 0

			p := NewProvider(config)
			faces, err := p.DetectFaces(context.Background(), []byte("test-image"))

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Len(t, faces, tt.wantCount)

			if tt.wantCount > 0 {
				// Verify confidence and quality are calculated
				assert.Greater(t, faces[0].Confidence, 0.0)
				assert.Greater(t, faces[0].QualityScore, 0.0)
			}
		})
	}
}

// TestProvider_IndexFace tests face indexing with mocked server
func TestProvider_IndexFace(t *testing.T) {
	tests := []struct {
		name           string
		serverResponse RepresentResponse
		serverStatus   int
		wantEmbLen     int
		wantErr        bool
		wantErrType    error
	}{
		{
			name: "successful indexing",
			serverResponse: RepresentResponse{
				Results: []RepresentResult{
					{
						Embedding:  make([]float64, 512),
						FacialArea: FacialArea{X: 10, Y: 20, W: 200, H: 200},
					},
				},
			},
			serverStatus: http.StatusOK,
			wantEmbLen:   512,
			wantErr:      false,
		},
		{
			name:           "no face in response",
			serverResponse: RepresentResponse{Results: []RepresentResult{}},
			serverStatus:   http.StatusOK,
			wantErr:        true,
			wantErrType:    ErrNoFaceInResponse,
		},
		{
			name:           "server error",
			serverResponse: RepresentResponse{},
			serverStatus:   http.StatusInternalServerError,
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.serverStatus)
				_ = json.NewEncoder(w).Encode(tt.serverResponse)
			}))
			defer server.Close()

			config := DefaultConfig()
			config.BaseURL = server.URL
			config.RetryCount = 0

			p := NewProvider(config)
			faceID, embedding, err := p.IndexFace(context.Background(), []byte("test-image"))

			if tt.wantErr {
				assert.Error(t, err)
				if tt.wantErrType != nil {
					assert.ErrorIs(t, err, tt.wantErrType)
				}
				return
			}

			require.NoError(t, err)
			assert.NotEmpty(t, faceID)
			assert.Len(t, embedding, tt.wantEmbLen)
		})
	}
}

// TestProvider_CompareFaces tests face comparison
func TestProvider_CompareFaces(t *testing.T) {
	p := NewProvider(DefaultConfig())

	tests := []struct {
		name       string
		embedding1 []float64
		embedding2 []float64
		wantMin    float64
		wantMax    float64
	}{
		{
			name:       "identical embeddings",
			embedding1: []float64{1.0, 0.0, 0.0},
			embedding2: []float64{1.0, 0.0, 0.0},
			wantMin:    0.99,
			wantMax:    1.0,
		},
		{
			name:       "different embeddings",
			embedding1: []float64{1.0, 0.0, 0.0},
			embedding2: []float64{0.0, 1.0, 0.0},
			wantMin:    -0.01,
			wantMax:    0.01,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity, err := p.CompareFaces(context.Background(), tt.embedding1, tt.embedding2)
			require.NoError(t, err)
			assert.GreaterOrEqual(t, similarity, tt.wantMin)
			assert.LessOrEqual(t, similarity, tt.wantMax)
		})
	}
}

// TestProvider_DeleteFace tests face deletion (no-op)
func TestProvider_DeleteFace(t *testing.T) {
	p := NewProvider(DefaultConfig())
	err := p.DeleteFace(context.Background(), "any-face-id")
	assert.NoError(t, err)
}

// TestCalculateConfidence tests confidence calculation
func TestCalculateConfidence(t *testing.T) {
	tests := []struct {
		name     string
		faceArea float64
		wantMin  float64
		wantMax  float64
	}{
		{
			name:     "very small face",
			faceArea: 1000, // 31x31 pixels
			wantMin:  0.49,
			wantMax:  0.51,
		},
		{
			name:     "minimum face area",
			faceArea: minFaceArea,
			wantMin:  0.69,
			wantMax:  0.71,
		},
		{
			name:     "medium face",
			faceArea: 40000, // 200x200 pixels
			wantMin:  0.73,
			wantMax:  0.77,
		},
		{
			name:     "large face",
			faceArea: maxFaceArea,
			wantMin:  0.98,
			wantMax:  1.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := calculateConfidence(tt.faceArea)
			assert.GreaterOrEqual(t, confidence, tt.wantMin)
			assert.LessOrEqual(t, confidence, tt.wantMax)
		})
	}
}

// TestCalculateQuality tests quality calculation
func TestCalculateQuality(t *testing.T) {
	tests := []struct {
		name     string
		faceArea float64
		wantMin  float64
		wantMax  float64
	}{
		{
			name:     "very small face",
			faceArea: 1000,
			wantMin:  0.39,
			wantMax:  0.41,
		},
		{
			name:     "minimum face area",
			faceArea: minFaceArea,
			wantMin:  0.59,
			wantMax:  0.61,
		},
		{
			name:     "large face",
			faceArea: maxFaceArea,
			wantMin:  0.94,
			wantMax:  0.96,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quality := calculateQuality(tt.faceArea)
			assert.GreaterOrEqual(t, quality, tt.wantMin)
			assert.LessOrEqual(t, quality, tt.wantMax)
		})
	}
}

// TestNormalizeEmbedding tests embedding normalization
func TestNormalizeEmbedding(t *testing.T) {
	tests := []struct {
		name      string
		embedding []float64
		wantNorm  float64
	}{
		{
			name:      "unit vector",
			embedding: []float64{1.0, 0.0, 0.0},
			wantNorm:  1.0,
		},
		{
			name:      "non-unit vector",
			embedding: []float64{3.0, 4.0},
			wantNorm:  1.0,
		},
		{
			name:      "empty vector",
			embedding: []float64{},
			wantNorm:  0.0, // special case
		},
		{
			name:      "zero vector",
			embedding: []float64{0.0, 0.0, 0.0},
			wantNorm:  0.0, // special case
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := NormalizeEmbedding(tt.embedding)

			if len(tt.embedding) == 0 || tt.wantNorm == 0.0 {
				assert.Equal(t, len(tt.embedding), len(normalized))
				return
			}

			// Calculate norm of normalized vector
			var norm float64
			for _, v := range normalized {
				norm += v * v
			}
			norm = abs(norm - 1.0)
			assert.Less(t, norm, 0.0001)
		})
	}
}
