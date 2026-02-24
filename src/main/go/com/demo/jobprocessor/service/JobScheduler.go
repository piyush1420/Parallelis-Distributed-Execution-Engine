package service

import (
	"context"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"

	"distributed-job-processor/config"
	"distributed-job-processor/model"
	"distributed-job-processor/repository"
)

// JobScheduler polls the database for PENDING jobs and publishes them to Kafka.
//
// Flow:
// 1. Every 5 seconds, query database for PENDING jobs (scheduled_at <= now)
// 2. For each job found:
//    a. Publish job ID to Kafka topic
//    b. Update job status to RUNNING
//    c. If Kafka publish fails, keep status as PENDING (retry next poll)
//
// This decouples the API (fast response) from job processing (slow).
type JobScheduler struct {
	jobRepository *repository.JobRepository
	kafkaWriter   *kafka.Writer
	pollInterval  time.Duration
	stopCh        chan struct{}
}

// NewJobScheduler creates a new JobScheduler with the given dependencies.
func NewJobScheduler(jobRepository *repository.JobRepository, kafkaWriter *kafka.Writer) *JobScheduler {
	interval := 5 * time.Second // default
	if val := os.Getenv("SCHEDULER_POLL_INTERVAL"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			interval = time.Duration(parsed) * time.Millisecond
		}
	}

	return &JobScheduler{
		jobRepository: jobRepository,
		kafkaWriter:   kafkaWriter,
		pollInterval:  interval,
		stopCh:        make(chan struct{}),
	}
}

// Start begins the scheduler polling loop in a goroutine.
// Equivalent to Spring's @Scheduled(fixedDelay).
// Fixed delay ensures we don't start next poll until previous completes.
// This prevents overwhelming the system during high load.
func (s *JobScheduler) Start() {
	// Job scheduling loop
	go func() {
		log.Printf("Job scheduler started (poll interval: %v)", s.pollInterval)
		for {
			select {
			case <-s.stopCh:
				log.Println("Job scheduler stopped")
				return
			default:
				s.scheduleJobs()
				time.Sleep(s.pollInterval)
			}
		}
	}()

	// Statistics logging loop (every 60 seconds)
	go func() {
		ticker := time.NewTicker(60 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-s.stopCh:
				return
			case <-ticker.C:
				s.LogStatistics()
			}
		}
	}()
}

// Stop gracefully stops the scheduler.
func (s *JobScheduler) Stop() {
	close(s.stopCh)
}

// scheduleJobs polls the database for PENDING jobs and publishes them to Kafka.
func (s *JobScheduler) scheduleJobs() {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Error in scheduler poll: %v", r)
		}
	}()

	// Find all PENDING jobs that are scheduled to run now or in the past
	pendingJobs, err := s.jobRepository.FindByStatusAndScheduledAtBefore(
		model.StatusPending,
		time.Now(),
	)
	if err != nil {
		log.Printf("Error finding pending jobs: %v", err)
		return
	}

	if len(pendingJobs) == 0 {
		log.Println("No pending jobs found")
		return
	}

	log.Printf("Found %d pending jobs to schedule", len(pendingJobs))

	// Process each job
	for _, job := range pendingJobs {
		func(j model.Job) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Failed to schedule job %s: %v", j.ID, r)
				}
			}()
			s.scheduleJob(&j)
		}(job)
	}
}

// scheduleJob publishes a single job to Kafka.
func (s *JobScheduler) scheduleJob(job *model.Job) {
	jobID := job.ID.String()

	log.Printf("Scheduling job: id=%s, type=%s, clientId=%s, attempt=%d",
		jobID, job.Type, job.ClientID, job.Attempts)

	// Publish job ID to Kafka
	// Use clientId as key for partition routing
	err := s.kafkaWriter.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(job.ClientID),
			Value: []byte(jobID),
		},
	)

	if err != nil {
		// Failure: Kafka send failed
		// Keep status as PENDING so it will be retried in next poll
		log.Printf("Failed to publish job %s to Kafka: %v", jobID, err)
		return
	}

	// Success: Kafka message sent
	log.Printf("Job %s published to Kafka", jobID)

	// Update job status to RUNNING
	job.Status = model.StatusRunning
	now := time.Now()
	job.UpdatedAt = now
	if err := s.jobRepository.Save(job); err != nil {
		log.Printf("Failed to update job %s status to RUNNING: %v", jobID, err)
	}
}

// LogStatistics logs the current job statistics.
// Useful for monitoring and alerting.
func (s *JobScheduler) LogStatistics() {
	pending, _ := s.jobRepository.CountByStatus(model.StatusPending)
	running, _ := s.jobRepository.CountByStatus(model.StatusRunning)
	completed, _ := s.jobRepository.CountByStatus(model.StatusCompleted)
	failed, _ := s.jobRepository.CountByStatus(model.StatusFailed)
	deadLetter, _ := s.jobRepository.CountByStatus(model.StatusDeadLetter)

	log.Printf("Job Statistics - PENDING: %d, RUNNING: %d, COMPLETED: %d, FAILED: %d, DEAD_LETTER: %d",
		pending, running, completed, failed, deadLetter)
}