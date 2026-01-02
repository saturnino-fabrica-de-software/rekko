package deepface

import "errors"

var (
	ErrDeepFaceUnavailable = errors.New("deepface service unavailable")
	ErrDeepFaceTimeout     = errors.New("deepface request timeout")
	ErrInvalidResponse     = errors.New("invalid response from deepface")
	ErrNoFaceInResponse    = errors.New("no face data in deepface response")
	ErrInvalidImageFormat  = errors.New("invalid image format for deepface")
)
