package exception

import (
	"fmt"

	"github.com/google/uuid"
)

// JobNotFoundError is returned when a requested job cannot be found in the system.
// Implements the error interface.
type JobNotFoundError struct {
	JobID uuid.UUID
}

// Error returns the error message string.
func (e *JobNotFoundError) Error() string {
	return fmt.Sprintf("Job not found with id: %s", e.JobID)
}

// NewJobNotFoundError creates a new JobNotFoundError for the given job ID.
func NewJobNotFoundError(jobID uuid.UUID) *JobNotFoundError {
	return &JobNotFoundError{JobID: jobID}
}

// IsJobNotFoundError checks if an error is a JobNotFoundError.
func IsJobNotFoundError(err error) bool {
	_, ok := err.(*JobNotFoundError)
	return ok
}