package deepface

import (
	"math"
)

// CosineSimilarity calculates the cosine similarity between two embedding vectors.
// Returns a value between -1.0 (opposite) and 1.0 (identical).
// For face embeddings, values > 0.8 typically indicate a match.
func CosineSimilarity(embedding1, embedding2 []float64) float64 {
	if len(embedding1) != len(embedding2) || len(embedding1) == 0 {
		return 0.0
	}

	var dotProduct, norm1, norm2 float64
	for i := range embedding1 {
		dotProduct += embedding1[i] * embedding2[i]
		norm1 += embedding1[i] * embedding1[i]
		norm2 += embedding2[i] * embedding2[i]
	}

	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

// NormalizeEmbedding normalizes an embedding vector to unit length.
// This is useful for consistent similarity calculations.
func NormalizeEmbedding(embedding []float64) []float64 {
	if len(embedding) == 0 {
		return embedding
	}

	var norm float64
	for _, v := range embedding {
		norm += v * v
	}

	if norm == 0 {
		return embedding
	}

	norm = math.Sqrt(norm)
	normalized := make([]float64, len(embedding))
	for i, v := range embedding {
		normalized[i] = v / norm
	}

	return normalized
}
