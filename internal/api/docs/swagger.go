package docs

import (
	"github.com/go-swagno/swagno"
	"github.com/go-swagno/swagno/components/endpoint"
	"github.com/go-swagno/swagno/components/http/response"
	"github.com/go-swagno/swagno/components/mime"
	"github.com/go-swagno/swagno/components/parameter"
)

// RegisterFaceResponse represents the response for a successful face registration
type RegisterFaceResponse struct {
	FaceID       string  `json:"face_id" example:"550e8400-e29b-41d4-a716-446655440000"`
	ExternalID   string  `json:"external_id" example:"user-123"`
	QualityScore float64 `json:"quality_score" example:"0.95"`
	CreatedAt    string  `json:"created_at" example:"2024-01-01T00:00:00Z"`
}

// VerifyFaceResponse represents the response for face verification
type VerifyFaceResponse struct {
	Verified   bool    `json:"verified" example:"true"`
	Confidence float64 `json:"confidence" example:"0.92"`
	ExternalID string  `json:"external_id" example:"user-123"`
	LatencyMs  int64   `json:"latency_ms" example:"45"`
}

// ErrorResponse represents a standard error response
type ErrorResponse struct {
	Code    string `json:"code" example:"VALIDATION_FAILED"`
	Message string `json:"message" example:"Request validation failed"`
}

// EmptyResponse represents no content response (204)
type EmptyResponse struct{}

// NewSwagger creates and configures the Swagger documentation
func NewSwagger() *swagno.Swagger {
	sw := swagno.New(swagno.Config{
		Title:       "Rekko Face Recognition API",
		Version:     "v1.0.0",
		Description: "FRaaS (Facial Recognition as a Service) API for event access control with multi-tenancy support",
		Host:        "localhost:3000",
		Path:        "/v1",
	})

	endpoints := []*endpoint.EndPoint{
		// POST /v1/faces - Register Face
		endpoint.New(
			endpoint.POST,
			"/faces",
			endpoint.WithTags("Faces"),
			endpoint.WithSummary("Register a new face"),
			endpoint.WithDescription("Registers a new face for a user identified by external_id. The image must contain exactly one face."),
			endpoint.WithConsume([]mime.MIME{mime.MULTIFORM}),
			endpoint.WithProduce([]mime.MIME{mime.JSON}),
			endpoint.WithParams(
				parameter.StrParam("external_id", parameter.Form, parameter.WithRequired(), parameter.WithDescription("Unique identifier for the user in your system")),
				parameter.FileParam("image", parameter.WithRequired(), parameter.WithDescription("Face image file (JPEG, PNG)")),
			),
			endpoint.WithSuccessfulReturns([]response.Response{
				response.New(RegisterFaceResponse{}, "201", "Face registered successfully"),
			}),
			endpoint.WithErrors([]response.Response{
				response.New(ErrorResponse{Code: "VALIDATION_FAILED", Message: "external_id is required"}, "400", "Bad Request"),
				response.New(ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid or missing API key"}, "401", "Unauthorized"),
				response.New(ErrorResponse{Code: "FACE_ALREADY_EXISTS", Message: "Face already registered"}, "409", "Conflict"),
				response.New(ErrorResponse{Code: "NO_FACE_DETECTED", Message: "No face detected in the image"}, "422", "Unprocessable Entity"),
				response.New(ErrorResponse{Code: "MULTIPLE_FACES", Message: "Multiple faces detected"}, "422", "Unprocessable Entity"),
				response.New(ErrorResponse{Code: "INTERNAL_ERROR", Message: "An unexpected error occurred"}, "500", "Internal Server Error"),
			}),
			endpoint.WithSecurity([]map[string][]string{{"ApiKeyAuth": {}}}),
		),

		// POST /v1/faces/verify - Verify Face
		endpoint.New(
			endpoint.POST,
			"/faces/verify",
			endpoint.WithTags("Faces"),
			endpoint.WithSummary("Verify a face (1:1 matching)"),
			endpoint.WithDescription("Verifies if the provided face image matches the registered face for the given external_id."),
			endpoint.WithConsume([]mime.MIME{mime.MULTIFORM}),
			endpoint.WithProduce([]mime.MIME{mime.JSON}),
			endpoint.WithParams(
				parameter.StrParam("external_id", parameter.Form, parameter.WithRequired(), parameter.WithDescription("Unique identifier of the user to verify")),
				parameter.FileParam("image", parameter.WithRequired(), parameter.WithDescription("Face image file to verify")),
			),
			endpoint.WithSuccessfulReturns([]response.Response{
				response.New(VerifyFaceResponse{}, "200", "Face verification completed"),
			}),
			endpoint.WithErrors([]response.Response{
				response.New(ErrorResponse{Code: "VALIDATION_FAILED", Message: "external_id is required"}, "400", "Bad Request"),
				response.New(ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid or missing API key"}, "401", "Unauthorized"),
				response.New(ErrorResponse{Code: "FACE_NOT_FOUND", Message: "Face not found"}, "404", "Not Found"),
				response.New(ErrorResponse{Code: "NO_FACE_DETECTED", Message: "No face detected"}, "422", "Unprocessable Entity"),
				response.New(ErrorResponse{Code: "MULTIPLE_FACES", Message: "Multiple faces detected"}, "422", "Unprocessable Entity"),
				response.New(ErrorResponse{Code: "INTERNAL_ERROR", Message: "An unexpected error occurred"}, "500", "Internal Server Error"),
			}),
			endpoint.WithSecurity([]map[string][]string{{"ApiKeyAuth": {}}}),
		),

		// DELETE /v1/faces/{external_id} - Delete Face
		endpoint.New(
			endpoint.DELETE,
			"/faces/{external_id}",
			endpoint.WithTags("Faces"),
			endpoint.WithSummary("Delete a registered face"),
			endpoint.WithDescription("Permanently deletes the face associated with the given external_id. LGPD compliant."),
			endpoint.WithProduce([]mime.MIME{mime.JSON}),
			endpoint.WithParams(
				parameter.StrParam("external_id", parameter.Path, parameter.WithRequired(), parameter.WithDescription("Unique identifier of the user whose face should be deleted")),
			),
			endpoint.WithSuccessfulReturns([]response.Response{
				response.New(EmptyResponse{}, "204", "Face deleted successfully"),
			}),
			endpoint.WithErrors([]response.Response{
				response.New(ErrorResponse{Code: "VALIDATION_FAILED", Message: "external_id is required"}, "400", "Bad Request"),
				response.New(ErrorResponse{Code: "UNAUTHORIZED", Message: "Invalid or missing API key"}, "401", "Unauthorized"),
				response.New(ErrorResponse{Code: "FACE_NOT_FOUND", Message: "Face not found"}, "404", "Not Found"),
				response.New(ErrorResponse{Code: "INTERNAL_ERROR", Message: "An unexpected error occurred"}, "500", "Internal Server Error"),
			}),
			endpoint.WithSecurity([]map[string][]string{{"ApiKeyAuth": {}}}),
		),
	}

	sw.AddEndpoints(endpoints)

	return sw
}
