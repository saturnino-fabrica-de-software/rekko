package deepface

import (
	"testing"

	"github.com/saturnino-fabrica-de-software/rekko/internal/provider"
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
