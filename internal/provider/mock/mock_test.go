package mock

import (
	"context"
	"testing"
)

func TestProvider_DetectFaces(t *testing.T) {
	p := New()
	ctx := context.Background()

	tests := []struct {
		name      string
		image     []byte
		wantFaces int
		wantErr   bool
	}{
		{
			name:      "valid image",
			image:     make([]byte, 5000),
			wantFaces: 1,
			wantErr:   false,
		},
		{
			name:      "image too small",
			image:     make([]byte, 100),
			wantFaces: 0,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			faces, err := p.DetectFaces(ctx, tt.image)
			if (err != nil) != tt.wantErr {
				t.Errorf("DetectFaces() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if len(faces) != tt.wantFaces {
				t.Errorf("DetectFaces() got %d faces, want %d", len(faces), tt.wantFaces)
			}
		})
	}
}

func TestProvider_IndexFace(t *testing.T) {
	p := New()
	ctx := context.Background()

	image := make([]byte, 5000)
	for i := range image {
		image[i] = byte(i % 256)
	}

	faceID, embedding, err := p.IndexFace(ctx, image)
	if err != nil {
		t.Fatalf("IndexFace() error = %v", err)
	}

	if faceID == "" {
		t.Error("IndexFace() returned empty faceID")
	}

	if len(embedding) != embeddingDimension {
		t.Errorf("IndexFace() embedding length = %d, want %d", len(embedding), embeddingDimension)
	}

	var norm float64
	for _, v := range embedding {
		norm += v * v
	}
	if norm < 0.99 || norm > 1.01 {
		t.Errorf("IndexFace() embedding not normalized, norm = %f", norm)
	}
}

func TestProvider_IndexFace_Deterministic(t *testing.T) {
	p := New()
	ctx := context.Background()

	image := []byte("test image content that is long enough to be valid")
	image = append(image, make([]byte, 1000)...)

	_, emb1, _ := p.IndexFace(ctx, image)
	_, emb2, _ := p.IndexFace(ctx, image)

	for i := range emb1 {
		if emb1[i] != emb2[i] {
			t.Error("IndexFace() should be deterministic for same input")
			break
		}
	}
}

func TestProvider_CompareFaces(t *testing.T) {
	p := New()
	ctx := context.Background()

	image1 := make([]byte, 5000)
	image2 := make([]byte, 5000)
	for i := range image1 {
		image1[i] = byte(i % 256)
		image2[i] = byte(i % 256)
	}

	_, emb1, _ := p.IndexFace(ctx, image1)
	_, emb2, _ := p.IndexFace(ctx, image2)

	similarity, err := p.CompareFaces(ctx, emb1, emb2)
	if err != nil {
		t.Fatalf("CompareFaces() error = %v", err)
	}

	if similarity < 0.99 {
		t.Errorf("CompareFaces() same image similarity = %f, want ~1.0", similarity)
	}

	image3 := make([]byte, 5000)
	for i := range image3 {
		image3[i] = byte((i * 7) % 256)
	}
	_, emb3, _ := p.IndexFace(ctx, image3)

	diffSimilarity, _ := p.CompareFaces(ctx, emb1, emb3)
	if diffSimilarity >= similarity {
		t.Errorf("CompareFaces() different images should have lower similarity")
	}
}

func TestProvider_DeleteFace(t *testing.T) {
	p := New()
	ctx := context.Background()

	err := p.DeleteFace(ctx, "any-face-id")
	if err != nil {
		t.Errorf("DeleteFace() error = %v, want nil", err)
	}
}
