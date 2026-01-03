package rekognition

import (
	"context"

	"github.com/aws/aws-sdk-go-v2/service/rekognition"
)

// mockRekognitionAPI is a mock implementation of RekognitionAPI interface for testing
type mockRekognitionAPI struct {
	detectFacesFunc        func(ctx context.Context, params *rekognition.DetectFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.DetectFacesOutput, error)
	indexFacesFunc         func(ctx context.Context, params *rekognition.IndexFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.IndexFacesOutput, error)
	deleteFacesFunc        func(ctx context.Context, params *rekognition.DeleteFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.DeleteFacesOutput, error)
	compareFacesFunc       func(ctx context.Context, params *rekognition.CompareFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.CompareFacesOutput, error)
	searchFacesByImageFunc func(ctx context.Context, params *rekognition.SearchFacesByImageInput, optFns ...func(*rekognition.Options)) (*rekognition.SearchFacesByImageOutput, error)
	createCollectionFunc   func(ctx context.Context, params *rekognition.CreateCollectionInput, optFns ...func(*rekognition.Options)) (*rekognition.CreateCollectionOutput, error)
	deleteCollectionFunc   func(ctx context.Context, params *rekognition.DeleteCollectionInput, optFns ...func(*rekognition.Options)) (*rekognition.DeleteCollectionOutput, error)
	describeCollectionFunc func(ctx context.Context, params *rekognition.DescribeCollectionInput, optFns ...func(*rekognition.Options)) (*rekognition.DescribeCollectionOutput, error)
	listCollectionsFunc    func(ctx context.Context, params *rekognition.ListCollectionsInput, optFns ...func(*rekognition.Options)) (*rekognition.ListCollectionsOutput, error)
}

func (m *mockRekognitionAPI) DetectFaces(ctx context.Context, params *rekognition.DetectFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.DetectFacesOutput, error) {
	if m.detectFacesFunc != nil {
		return m.detectFacesFunc(ctx, params, optFns...)
	}
	return &rekognition.DetectFacesOutput{}, nil
}

func (m *mockRekognitionAPI) IndexFaces(ctx context.Context, params *rekognition.IndexFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.IndexFacesOutput, error) {
	if m.indexFacesFunc != nil {
		return m.indexFacesFunc(ctx, params, optFns...)
	}
	return &rekognition.IndexFacesOutput{}, nil
}

func (m *mockRekognitionAPI) DeleteFaces(ctx context.Context, params *rekognition.DeleteFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.DeleteFacesOutput, error) {
	if m.deleteFacesFunc != nil {
		return m.deleteFacesFunc(ctx, params, optFns...)
	}
	return &rekognition.DeleteFacesOutput{}, nil
}

func (m *mockRekognitionAPI) CompareFaces(ctx context.Context, params *rekognition.CompareFacesInput, optFns ...func(*rekognition.Options)) (*rekognition.CompareFacesOutput, error) {
	if m.compareFacesFunc != nil {
		return m.compareFacesFunc(ctx, params, optFns...)
	}
	return &rekognition.CompareFacesOutput{}, nil
}

func (m *mockRekognitionAPI) SearchFacesByImage(ctx context.Context, params *rekognition.SearchFacesByImageInput, optFns ...func(*rekognition.Options)) (*rekognition.SearchFacesByImageOutput, error) {
	if m.searchFacesByImageFunc != nil {
		return m.searchFacesByImageFunc(ctx, params, optFns...)
	}
	return &rekognition.SearchFacesByImageOutput{}, nil
}

func (m *mockRekognitionAPI) CreateCollection(ctx context.Context, params *rekognition.CreateCollectionInput, optFns ...func(*rekognition.Options)) (*rekognition.CreateCollectionOutput, error) {
	if m.createCollectionFunc != nil {
		return m.createCollectionFunc(ctx, params, optFns...)
	}
	return &rekognition.CreateCollectionOutput{}, nil
}

func (m *mockRekognitionAPI) DeleteCollection(ctx context.Context, params *rekognition.DeleteCollectionInput, optFns ...func(*rekognition.Options)) (*rekognition.DeleteCollectionOutput, error) {
	if m.deleteCollectionFunc != nil {
		return m.deleteCollectionFunc(ctx, params, optFns...)
	}
	return &rekognition.DeleteCollectionOutput{}, nil
}

func (m *mockRekognitionAPI) DescribeCollection(ctx context.Context, params *rekognition.DescribeCollectionInput, optFns ...func(*rekognition.Options)) (*rekognition.DescribeCollectionOutput, error) {
	if m.describeCollectionFunc != nil {
		return m.describeCollectionFunc(ctx, params, optFns...)
	}
	return &rekognition.DescribeCollectionOutput{}, nil
}

func (m *mockRekognitionAPI) ListCollections(ctx context.Context, params *rekognition.ListCollectionsInput, optFns ...func(*rekognition.Options)) (*rekognition.ListCollectionsOutput, error) {
	if m.listCollectionsFunc != nil {
		return m.listCollectionsFunc(ctx, params, optFns...)
	}
	return &rekognition.ListCollectionsOutput{}, nil
}
