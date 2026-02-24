package dto

import (
	"time"

	"github.com/google/uuid"

	"distributed-job-processor/model"
)

// JobResponse is the response DTO for job information.
// Returned when creating a job or querying job status.
// Fields with omitempty mirror Java's @JsonInclude(NON_NULL).
type JobResponse struct {
	JobID        uuid.UUID       `json:"jobId"`
	ClientID     string          `json:"clientId"`
	Type         model.JobType   `json:"type"`
	Status       model.JobStatus `json:"status"`
	Payload      string          `json:"payload"`
	Attempts     int             `json:"attempts"`
	MaxRetries   int             `json:"maxRetries"`
	CreatedAt    time.Time       `json:"createdAt"`
	ScheduledAt  *time.Time      `json:"scheduledAt,omitempty"`
	CompletedAt  *time.Time      `json:"completedAt,omitempty"`
	ErrorMessage *string         `json:"errorMessage,omitempty"`
}

// JobResponseFrom converts a Job entity to a JobResponse DTO.
func JobResponseFrom(job *model.Job) JobResponse {
	return JobResponse{
		JobID:        job.ID,
		ClientID:     job.ClientID,
		Type:         job.Type,
		Status:       job.Status,
		Payload:      job.Payload,
		Attempts:     job.Attempts,
		MaxRetries:   job.MaxRetries,
		CreatedAt:    job.CreatedAt,
		ScheduledAt:  job.ScheduledAt,
		CompletedAt:  job.CompletedAt,
		ErrorMessage: job.ErrorMessage,
	}
}

// JobResponseMinimal creates a minimal response with just the essential fields.
// Used for job creation response (202 Accepted).
func JobResponseMinimal(job *model.Job) JobResponse {
	return JobResponse{
		JobID:      job.ID,
		ClientID:   job.ClientID,
		Type:       job.Type,
		Status:     job.Status,
		Payload:    job.Payload,
		Attempts:   job.Attempts,
		MaxRetries: job.MaxRetries,
		CreatedAt:  job.CreatedAt,
	}
}