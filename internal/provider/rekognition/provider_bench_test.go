package rekognition

import (
	"context"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/rekognition"
	"github.com/aws/aws-sdk-go-v2/service/rekognition/types"
	"github.com/google/uuid"
)

// BenchmarkDetectFaces benchmarks face detection operation
// Target: P99 < 200ms (excluding AWS network latency)
// This measures local overhead only (using mock)
func BenchmarkDetectFaces(b *testing.B) {
	mock := &mockRekognitionAPI{
		detectFacesFunc: func(ctx context.Context, params *rekognition.DetectFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.DetectFacesOutput, error) {
			return &rekognition.DetectFacesOutput{
				FaceDetails: []types.FaceDetail{
					{
						BoundingBox: &types.BoundingBox{
							Left:   aws.Float32(0.1),
							Top:    aws.Float32(0.2),
							Width:  aws.Float32(0.3),
							Height: aws.Float32(0.4),
						},
						Confidence: aws.Float32(99.5),
						Quality: &types.ImageQuality{
							Brightness: aws.Float32(80),
							Sharpness:  aws.Float32(90),
						},
					},
				},
			}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}
	imageData := make([]byte, 1024) // 1KB fake image
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = provider.DetectFaces(ctx, imageData)
	}
}

// BenchmarkIndexFace benchmarks face indexing operation
// This is a critical operation for registration flow
func BenchmarkIndexFace(b *testing.B) {
	mock := &mockRekognitionAPI{
		indexFacesFunc: func(ctx context.Context, params *rekognition.IndexFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.IndexFacesOutput, error) {
			return &rekognition.IndexFacesOutput{
				FaceRecords: []types.FaceRecord{
					{
						Face: &types.Face{
							FaceId:     aws.String("face-123"),
							Confidence: aws.Float32(99.9),
						},
					},
				},
			}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}
	imageData := make([]byte, 1024)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, _ = provider.IndexFace(ctx, imageData)
	}
}

// BenchmarkSearchFacesByImage benchmarks 1:N face search
// CRITICAL: This is the most performance-sensitive operation
// Used in event entry flow where P99 < 200ms is target
func BenchmarkSearchFacesByImage(b *testing.B) {
	mock := &mockRekognitionAPI{
		searchFacesByImageFunc: func(ctx context.Context, params *rekognition.SearchFacesByImageInput, optFns ...func(*rekognition.Options)) (*rekognition.SearchFacesByImageOutput, error) {
			return &rekognition.SearchFacesByImageOutput{
				FaceMatches: []types.FaceMatch{
					{
						Face: &types.Face{
							FaceId:          aws.String("face-123"),
							ExternalImageId: aws.String("user-456"),
							Confidence:      aws.Float32(99.5),
						},
						Similarity: aws.Float32(98.5),
					},
				},
			}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}
	imageData := make([]byte, 1024)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = provider.SearchFacesByImage(ctx, imageData, 10, 0.8)
	}
}

// BenchmarkCompareFaceImages benchmarks 1:1 face comparison
// Used for verification flow
func BenchmarkCompareFaceImages(b *testing.B) {
	mock := &mockRekognitionAPI{
		compareFacesFunc: func(ctx context.Context, params *rekognition.CompareFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.CompareFacesOutput, error) {
			return &rekognition.CompareFacesOutput{
				FaceMatches: []types.CompareFacesMatch{
					{Similarity: aws.Float32(98.5)},
				},
			}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}
	sourceImage := make([]byte, 1024)
	targetImage := make([]byte, 1024)
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = provider.CompareFaceImages(ctx, sourceImage, targetImage, 0.8)
	}
}

// BenchmarkValidateImage benchmarks image validation
// This should be extremely fast (< 1μs)
func BenchmarkValidateImage(b *testing.B) {
	imageData := make([]byte, 1024*1024) // 1MB image

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = validateImage(imageData)
	}
}

// BenchmarkCalculateQualityScore benchmarks quality score calculation
// Should be zero-allocation and extremely fast
func BenchmarkCalculateQualityScore(b *testing.B) {
	provider := &Provider{}
	quality := &types.ImageQuality{
		Brightness: aws.Float32(80.0),
		Sharpness:  aws.Float32(90.0),
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = provider.calculateQualityScore(quality)
	}
}

// BenchmarkDetectFaces_ImageSizes benchmarks impact of image size on detection
// Measures how image size affects performance (primarily memory allocation)
func BenchmarkDetectFaces_ImageSizes(b *testing.B) {
	sizes := []struct {
		name string
		size int
	}{
		{"1KB", 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
		{"5MB", 5 * 1024 * 1024},
	}

	for _, s := range sizes {
		b.Run(s.name, func(b *testing.B) {
			mock := &mockRekognitionAPI{
				detectFacesFunc: func(ctx context.Context, params *rekognition.DetectFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.DetectFacesOutput, error) {
					return &rekognition.DetectFacesOutput{
						FaceDetails: []types.FaceDetail{
							{
								BoundingBox: &types.BoundingBox{
									Left:   aws.Float32(0.1),
									Top:    aws.Float32(0.2),
									Width:  aws.Float32(0.3),
									Height: aws.Float32(0.4),
								},
								Confidence: aws.Float32(99.5),
								Quality: &types.ImageQuality{
									Brightness: aws.Float32(80),
									Sharpness:  aws.Float32(90),
								},
							},
						},
					}, nil
				},
			}

			client := &Client{rekognition: mock, config: DefaultConfig()}
			provider := &Provider{client: client, tenantID: uuid.New()}
			imageData := make([]byte, s.size)
			ctx := context.Background()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, _ = provider.DetectFaces(ctx, imageData)
			}
		})
	}
}

// BenchmarkSearchFacesByImage_ResultSizes benchmarks impact of result count
// Tests how the number of matches affects performance
func BenchmarkSearchFacesByImage_ResultSizes(b *testing.B) {
	resultCounts := []struct {
		name  string
		count int
	}{
		{"NoResults", 0},
		{"1Result", 1},
		{"10Results", 10},
		{"100Results", 100},
	}

	for _, rc := range resultCounts {
		b.Run(rc.name, func(b *testing.B) {
			// Generate mock face matches
			faceMatches := make([]types.FaceMatch, rc.count)
			for i := 0; i < rc.count; i++ {
				faceMatches[i] = types.FaceMatch{
					Face: &types.Face{
						FaceId:          aws.String("face-123"),
						ExternalImageId: aws.String("user-456"),
						Confidence:      aws.Float32(99.5),
					},
					Similarity: aws.Float32(98.5),
				}
			}

			mock := &mockRekognitionAPI{
				searchFacesByImageFunc: func(ctx context.Context, params *rekognition.SearchFacesByImageInput, optFns ...func(*rekognition.Options)) (*rekognition.SearchFacesByImageOutput, error) {
					return &rekognition.SearchFacesByImageOutput{
						FaceMatches: faceMatches,
					}, nil
				},
			}

			client := &Client{rekognition: mock, config: DefaultConfig()}
			provider := &Provider{client: client, tenantID: uuid.New()}
			imageData := make([]byte, 1024)
			ctx := context.Background()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, _ = provider.SearchFacesByImage(ctx, imageData, 100, 0.8)
			}
		})
	}
}

// BenchmarkDeleteFace benchmarks face deletion operation
func BenchmarkDeleteFace(b *testing.B) {
	faceID := "face-123"
	mock := &mockRekognitionAPI{
		deleteFacesFunc: func(ctx context.Context, params *rekognition.DeleteFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.DeleteFacesOutput, error) {
			return &rekognition.DeleteFacesOutput{
				DeletedFaces: []string{faceID},
			}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = provider.DeleteFace(ctx, faceID)
	}
}

// BenchmarkDetectFaces_MultipleFaces benchmarks detection with multiple faces
// Tests allocation efficiency when processing multiple faces
func BenchmarkDetectFaces_MultipleFaces(b *testing.B) {
	faceCounts := []struct {
		name  string
		count int
	}{
		{"1Face", 1},
		{"5Faces", 5},
		{"10Faces", 10},
		{"50Faces", 50},
	}

	for _, fc := range faceCounts {
		b.Run(fc.name, func(b *testing.B) {
			// Generate mock face details
			faceDetails := make([]types.FaceDetail, fc.count)
			for i := 0; i < fc.count; i++ {
				faceDetails[i] = types.FaceDetail{
					BoundingBox: &types.BoundingBox{
						Left:   aws.Float32(0.1),
						Top:    aws.Float32(0.2),
						Width:  aws.Float32(0.3),
						Height: aws.Float32(0.4),
					},
					Confidence: aws.Float32(99.5),
					Quality: &types.ImageQuality{
						Brightness: aws.Float32(80),
						Sharpness:  aws.Float32(90),
					},
				}
			}

			mock := &mockRekognitionAPI{
				detectFacesFunc: func(ctx context.Context, params *rekognition.DetectFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.DetectFacesOutput, error) {
					return &rekognition.DetectFacesOutput{
						FaceDetails: faceDetails,
					}, nil
				},
			}

			client := &Client{rekognition: mock, config: DefaultConfig()}
			provider := &Provider{client: client, tenantID: uuid.New()}
			imageData := make([]byte, 1024)
			ctx := context.Background()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, _ = provider.DetectFaces(ctx, imageData)
			}
		})
	}
}

// BenchmarkGetFaceCount benchmarks retrieving face count from collection
func BenchmarkGetFaceCount(b *testing.B) {
	mock := &mockRekognitionAPI{
		describeCollectionFunc: func(ctx context.Context, params *rekognition.DescribeCollectionInput, optFns ...func(*rekognition.Options)) (*rekognition.DescribeCollectionOutput, error) {
			return &rekognition.DescribeCollectionOutput{
				FaceCount: aws.Int64(1000),
			}, nil
		},
	}

	client := &Client{rekognition: mock, config: DefaultConfig()}
	provider := &Provider{client: client, tenantID: uuid.New()}
	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _ = provider.GetFaceCount(ctx)
	}
}

// BenchmarkCompareFaceImages_VariousSimilarities benchmarks comparison with different similarity thresholds
func BenchmarkCompareFaceImages_VariousSimilarities(b *testing.B) {
	thresholds := []struct {
		name      string
		threshold float64
	}{
		{"Threshold50", 0.5},
		{"Threshold70", 0.7},
		{"Threshold80", 0.8},
		{"Threshold90", 0.9},
		{"Threshold95", 0.95},
	}

	for _, th := range thresholds {
		b.Run(th.name, func(b *testing.B) {
			mock := &mockRekognitionAPI{
				compareFacesFunc: func(ctx context.Context, params *rekognition.CompareFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.CompareFacesOutput, error) {
					return &rekognition.CompareFacesOutput{
						FaceMatches: []types.CompareFacesMatch{
							{Similarity: aws.Float32(98.5)},
						},
					}, nil
				},
			}

			client := &Client{rekognition: mock, config: DefaultConfig()}
			provider := &Provider{client: client, tenantID: uuid.New()}
			sourceImage := make([]byte, 1024)
			targetImage := make([]byte, 1024)
			ctx := context.Background()

			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < b.N; i++ {
				_, _ = provider.CompareFaceImages(ctx, sourceImage, targetImage, th.threshold)
			}
		})
	}
}
