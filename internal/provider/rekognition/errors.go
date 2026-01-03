package rekognition

import "errors"

var (
	// ErrCollectionNotFound indicates that the specified collection does not exist
	ErrCollectionNotFound = errors.New("rekognition collection not found")

	// ErrCollectionAlreadyExists indicates that a collection with the same name already exists
	ErrCollectionAlreadyExists = errors.New("rekognition collection already exists")

	// ErrInvalidCredentials indicates that AWS credentials are invalid or missing
	ErrInvalidCredentials = errors.New("invalid or missing AWS credentials")

	// ErrNoFaceDetected indicates that no face was found in the provided image
	ErrNoFaceDetected = errors.New("no face detected in image")

	// ErrMultipleFaces indicates that multiple faces were detected when only one was expected
	ErrMultipleFaces = errors.New("multiple faces detected in image")

	// ErrFaceNotFound indicates that the specified face ID was not found in the collection
	ErrFaceNotFound = errors.New("face not found in rekognition collection")
)
