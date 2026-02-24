package exception

import "time"

// ErrorResponse is the standard error response format for API errors.
// Fields with omitempty mirror Java's @JsonInclude(NON_NULL).
type ErrorResponse struct {
	// Timestamp when the error occurred
	Timestamp time.Time `json:"timestamp"`

	// HTTP status code
	Status int `json:"status"`

	// Error type/category
	Error string `json:"error"`

	// Detailed error message
	Message string `json:"message"`

	// Validation errors (field name -> error message)
	// Only present for validation failures
	ValidationErrors map[string]string `json:"validationErrors,omitempty"`
}

// NewErrorResponse creates a new ErrorResponse with the current timestamp.
func NewErrorResponse(status int, err string, message string) ErrorResponse {
	return ErrorResponse{
		Timestamp: time.Now(),
		Status:    status,
		Error:     err,
		Message:   message,
	}
}

// NewValidationErrorResponse creates an ErrorResponse with validation errors.
func NewValidationErrorResponse(status int, err string, message string, validationErrors map[string]string) ErrorResponse {
	return ErrorResponse{
		Timestamp:        time.Now(),
		Status:           status,
		Error:            err,
		Message:          message,
		ValidationErrors: validationErrors,
	}
}