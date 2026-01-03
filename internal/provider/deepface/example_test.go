package deepface_test

import (
	"context"
	"fmt"
	"log"

	"github.com/saturnino-fabrica-de-software/rekko/internal/provider/deepface"
)

func ExampleProvider_DetectFaces() {
	// Create provider with default config
	config := deepface.DefaultConfig()
	provider := deepface.NewProvider(config)

	// Image bytes (in practice, load from file or HTTP request)
	var imageBytes []byte

	// Detect faces
	faces, err := provider.DetectFaces(context.Background(), imageBytes)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Detected %d faces\n", len(faces))
	for i, face := range faces {
		fmt.Printf("Face %d: confidence=%.2f, quality=%.2f\n",
			i, face.Confidence, face.QualityScore)
	}
}

func ExampleProvider_IndexFace() {
	// Create provider
	config := deepface.DefaultConfig()
	provider := deepface.NewProvider(config)

	// Image bytes
	var imageBytes []byte

	// Index face (extract embedding)
	faceID, embedding, err := provider.IndexFace(context.Background(), imageBytes)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Face indexed: ID=%s, embedding_size=%d\n", faceID, len(embedding))
}

func ExampleProvider_CompareFaces() {
	// Create provider
	config := deepface.DefaultConfig()
	provider := deepface.NewProvider(config)

	// Two embeddings to compare (typically from IndexFace)
	embedding1 := make([]float64, 512)
	embedding2 := make([]float64, 512)

	// Compare embeddings
	similarity, err := provider.CompareFaces(context.Background(), embedding1, embedding2)
	if err != nil {
		log.Fatal(err)
	}

	threshold := 0.8
	if similarity >= threshold {
		fmt.Printf("MATCH: similarity=%.4f (>= %.2f)\n", similarity, threshold)
	} else {
		fmt.Printf("NO MATCH: similarity=%.4f (< %.2f)\n", similarity, threshold)
	}
}
