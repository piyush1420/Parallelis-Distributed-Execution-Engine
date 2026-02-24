package service

import (
	"log"
	"time"

	"github.com/google/uuid"

	"distributed-job-processor/dto"
	"distributed-job-processor/exception"
	"distributed-job-processor/model"
	"distributed-job-processor/repository"
)

// JobService handles business logic for creating, retrieving, and updating jobs.
type JobService struct {
	jobRepository *repository.JobRepository
}

// NewJobService creates a new JobService with the given repository.
func NewJobService(jobRepository *repository.JobRepository) *JobService {
	return &JobService{jobRepository: jobRepository}
}

// CreateJob creates a new job from a request.
// The job is initially created in PENDING status and scheduled for immediate processing.
func (s *JobService) CreateJob(clientID string, request *dto.JobRequest) (*model.Job, error) {
	log.Printf("Creating new job for client: %s, type: %s", clientID, request.Type)

	now := time.Now()
	job := &model.Job{
		ID:         uuid.New(),
		ClientID:   clientID,
		Type:       request.Type,
		Status:     model.StatusPending,
		Payload:    request.Payload,
		Attempts:   0,
		MaxRetries: 3,
		CreatedAt:  now,
		ScheduledAt: &now, // Schedule immediately
	}

	if err := s.jobRepository.Save(job); err != nil {
		log.Printf("Failed to create job: %v", err)
		return nil, err
	}

	log.Printf("Job created successfully: id=%s, clientId=%s, type=%s",
		job.ID, job.ClientID, job.Type)

	return job, nil
}

// GetJob retrieves a job by its ID.
// Returns JobNotFoundError if the job does not exist.
func (s *JobService) GetJob(jobID uuid.UUID) (*model.Job, error) {
	log.Printf("Retrieving job: %s", jobID)

	job, err := s.jobRepository.FindByID(jobID)
	if err != nil {
		return nil, exception.NewJobNotFoundError(jobID)
	}

	return job, nil
}

// GetJobsByClient returns all jobs for a specific client.
// Useful for client-specific analytics and tracking.
func (s *JobService) GetJobsByClient(clientID string) ([]model.Job, error) {
	log.Printf("Retrieving jobs for client: %s", clientID)
	return s.jobRepository.FindByClientID(clientID)
}

// GetJobsByStatus returns all jobs with a specific status.
// Useful for monitoring and dashboards.
func (s *JobService) GetJobsByStatus(status model.JobStatus) ([]model.Job, error) {
	log.Printf("Retrieving jobs with status: %s", status)
	return s.jobRepository.FindByStatus(status)
}

// UpdateJobStatus updates the status of a job.
// This method is primarily used by the scheduler and workers.
// Returns JobNotFoundError if the job does not exist.
func (s *JobService) UpdateJobStatus(jobID uuid.UUID, newStatus model.JobStatus) (*model.Job, error) {
	log.Printf("Updating job status: id=%s, newStatus=%s", jobID, newStatus)

	job, err := s.GetJob(jobID)
	if err != nil {
		return nil, err
	}

	oldStatus := job.Status
	job.Status = newStatus

	// If job is completed or moved to dead letter, set completion timestamp
	if newStatus == model.StatusCompleted || newStatus == model.StatusDeadLetter {
		now := time.Now()
		job.CompletedAt = &now
	}

	if err := s.jobRepository.Save(job); err != nil {
		log.Printf("Failed to update job status: %v", err)
		return nil, err
	}

	log.Printf("Job status updated: id=%s, oldStatus=%s, newStatus=%s",
		jobID, oldStatus, newStatus)

	return job, nil
}

// CountJobsByStatus returns the count of jobs by status.
// Useful for dashboard metrics.
func (s *JobService) CountJobsByStatus(status model.JobStatus) int64 {
	count, err := s.jobRepository.CountByStatus(status)
	if err != nil {
		log.Printf("Error counting jobs by status %s: %v", status, err)
		return 0
	}
	return count
}

// FindJobsReadyForScheduling finds jobs that are ready to be scheduled.
// These are jobs in PENDING status that are scheduled to run now or in the past.
// This method is called by the scheduler component.
func (s *JobService) FindJobsReadyForScheduling() ([]model.Job, error) {
	return s.jobRepository.FindByStatusAndScheduledAtBefore(
		model.StatusPending,
		time.Now(),
	)
}

// FindStuckJobs finds jobs that appear to be stuck (running for too long).
// These jobs may need manual intervention.
func (s *JobService) FindStuckJobs(minutes int) ([]model.Job, error) {
	threshold := time.Now().Add(-time.Duration(minutes) * time.Minute)
	return s.jobRepository.FindStuckJobs(model.StatusRunning, threshold)
}