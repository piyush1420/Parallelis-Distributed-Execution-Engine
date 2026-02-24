package exception

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

// GlobalExceptionHandler is a Gin middleware that converts panics and errors
// to appropriate HTTP responses with error details.
// Equivalent to Spring's @RestControllerAdvice.

// ErrorHandlerMiddleware catches panics and returns proper error responses.
// Use as: r.Use(ErrorHandlerMiddleware())
func ErrorHandlerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.Printf("Recovered from panic: %v", err)

				// Handle all other exceptions - return 500 Internal Server Error
				response := NewErrorResponse(
					http.StatusInternalServerError,
					"Internal Server Error",
					"An unexpected error occurred",
				)
				c.JSON(http.StatusInternalServerError, response)
				c.Abort()
			}
		}()
		c.Next()
	}
}

// HandleJobNotFound returns a 404 Not Found response for missing jobs.
// Equivalent to Java's @ExceptionHandler(JobNotFoundException.class)
func HandleJobNotFound(c *gin.Context, message string) {
	response := NewErrorResponse(
		http.StatusNotFound,
		"Job Not Found",
		message,
	)
	c.JSON(http.StatusNotFound, response)
}

// HandleValidationError returns a 400 Bad Request response for validation failures.
// Equivalent to Java's @ExceptionHandler(MethodArgumentNotValidException.class)
func HandleValidationError(c *gin.Context, err error) {
	validationErrors := make(map[string]string)

	// Extract field-level validation errors from Gin's validator
	if ve, ok := err.(validator.ValidationErrors); ok {
		for _, fe := range ve {
			validationErrors[fe.Field()] = fe.Tag() + " validation failed"
		}
	}

	response := NewValidationErrorResponse(
		http.StatusBadRequest,
		"Validation Failed",
		"Invalid request parameters",
		validationErrors,
	)
	c.JSON(http.StatusBadRequest, response)
}

// HandleInternalError returns a 500 Internal Server Error response.
// Equivalent to Java's @ExceptionHandler(Exception.class)
func HandleInternalError(c *gin.Context) {
	response := NewErrorResponse(
		http.StatusInternalServerError,
		"Internal Server Error",
		"An unexpected error occurred",
	)
	c.JSON(http.StatusInternalServerError, response)
}