package face_test

import (
	"context"
	"fmt"
	"log"

	"github.com/google/uuid"

	"github.com/saturnino-fabrica-de-software/rekko/internal/config"
	"github.com/saturnino-fabrica-de-software/rekko/internal/face"
)

// ExampleNewFaceProvider_deepface demonstrates how to create a DeepFace provider
func ExampleNewFaceProvider_deepface() {
	ctx := context.Background()

	// Configuration for DeepFace (default provider)
	cfg := &config.Config{
		FaceProvider: "deepface",
		DeepFaceURL:  "http://localhost:5000",
	}

	tenantID := uuid.New()

	provider, err := face.NewFaceProvider(ctx, cfg, tenantID)
	if err != nil {
		log.Fatalf("failed to create provider: %v", err)
	}

	// Use provider to detect faces
	imageData := []byte("...") // Your image data here
	faces, err := provider.DetectFaces(ctx, imageData)
	if err != nil {
		log.Fatalf("failed to detect faces: %v", err)
	}

	fmt.Printf("Detected %d faces\n", len(faces))
}

// ExampleNewFaceProvider_rekognition demonstrates how to create a Rekognition provider
func ExampleNewFaceProvider_rekognition() {
	ctx := context.Background()

	// Configuration for AWS Rekognition
	// Requires AWS credentials via environment variables:
	// - AWS_ACCESS_KEY_ID
	// - AWS_SECRET_ACCESS_KEY
	cfg := &config.Config{
		FaceProvider: "rekognition",
		AWSRegion:    "us-east-1",
	}

	tenantID := uuid.New()

	provider, err := face.NewFaceProvider(ctx, cfg, tenantID)
	if err != nil {
		log.Fatalf("failed to create provider: %v", err)
	}

	// Use provider to index face
	imageData := []byte("...") // Your image data here
	faceID, embedding, err := provider.IndexFace(ctx, imageData)
	if err != nil {
		log.Fatalf("failed to index face: %v", err)
	}

	fmt.Printf("Indexed face with ID: %s, embedding size: %d\n", faceID, len(embedding))
}

// ExampleNewFaceProvider_environmentBased demonstrates runtime provider selection
func ExampleNewFaceProvider_environmentBased() {
	ctx := context.Background()

	// Load configuration from environment
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	tenantID := uuid.New()

	// Provider is selected based on FACE_PROVIDER env var
	// - "deepface" -> DeepFace provider
	// - "rekognition" -> AWS Rekognition provider
	// - empty or not set -> defaults to DeepFace
	provider, err := face.NewFaceProvider(ctx, cfg, tenantID)
	if err != nil {
		log.Fatalf("failed to create provider: %v", err)
	}

	// Use provider transparently (same interface regardless of implementation)
	imageData := []byte("...") // Your image data here
	faces, err := provider.DetectFaces(ctx, imageData)
	if err != nil {
		log.Fatalf("failed to detect faces: %v", err)
	}

	fmt.Printf("Using provider in %s environment, detected %d faces\n", cfg.Environment, len(faces))
}
