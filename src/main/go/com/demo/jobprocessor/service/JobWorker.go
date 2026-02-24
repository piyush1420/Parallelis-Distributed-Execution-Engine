package service

import (
	"context"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/segmentio/kafka-go"

	"distributed-job-processor/config"
	"distributed-job-processor/model"
	"distributed-job-processor/repository"
)

// JobWorker consumes jobs from Kafka and processes them.
//
// Flow:
// 1. Consume job ID from Kafka
// 2. Check Redis cache for job details (cache-aside pattern)
// 3. If cache miss, fetch from database and cache result
// 4. Process job based on type (simulate with time.Sleep)
// 5. Update job status to COMPLETED
// 6. Update cache
// 7. Acknowledge Kafka message (commit offset)
//
// Error Handling (Retry Logic with Exponential Backoff):
// - On failure: Increment attempts counter
// - If attempts < maxRetries:
//   - Set status back to PENDING
//   - Set scheduledAt = now + 2^attempts seconds (exponential backoff)
//   - Scheduler will pick it up again later
// - If attempts >= maxRetries:
//   - Set status to DEAD_LETTER
//   - Job will not be retried automatically
//
// Simulated Processing Times:
// - PAYMENT_PROCESS: 2 seconds (simulates Stripe API call)
// - EMAIL_CONFIRMATION: 1 second (simulates SendGrid API call)
type JobWorker struct {
	jobRepository *repository.JobRepository
	cacheService  *CacheService
	kafkaReader   *kafka.Reader
	concurrency   int
	stopCh        chan struct{}
}

// NewJobWorker creates a new JobWorker with the given dependencies.
func NewJobWorker(jobRepository *repository.JobRepository, cacheService *CacheService, concurrency int) *JobWorker {
	reader := config.NewKafkaConsumerReader(config.GetJobQueueTopic())

	return &JobWorker{
		jobRepository: jobRepository,
		cacheService:  cacheService,
		kafkaReader:   reader,
		concurrency:   concurrency,
		stopCh:        make(chan struct{}),
	}
}

// Start begins consuming messages from Kafka with the configured concurrency.
// Equivalent to Spring's @KafkaListener with setConcurrency(4).
// Multiple goroutines consume from the same reader (Kafka handles partition assignment).
func (w *JobWorker) Start() {
	log.Printf("Job worker started with concurrency: %d", w.concurrency)

	for i := 0; i < w.concurrency; i++ {
		go w.consumeLoop(i)
	}
}

// Stop gracefully stops the worker.
func (w *JobWorker) Stop() {
	close(w.stopCh)
	if err := w.kafkaReader.Close(); err != nil {
		log.Printf("Error closing Kafka reader: %v", err)
	}
}

// consumeLoop is the main consume loop for a single worker goroutine.
func (w *JobWorker) consumeLoop(workerID int) {
	log.Printf("Worker goroutine %d started", workerID)

	for {
		select {
		case <-w.stopCh:
			log.Printf("Worker goroutine %d stopped", workerID)
			return
		default:
			msg, err := w.kafkaReader.FetchMessage(context.Background())
			if err != nil {
				log.Printf("Worker %d: Error fetching message: %v", workerID, err)
				time.Sleep(1 * time.Second)
				continue
			}

			w.processJob(msg, workerID)
		}
	}
}

// processJob processes a single job message from Kafka.
//
// Configuration:
// - Manual acknowledgment: Only ack after successful DB update
// - Consumer group: "job-workers" (enables parallel processing)
// - Multiple instances can run in parallel
func (w *JobWorker) processJob(msg kafka.Message, workerID int) {
	jobIDStr := string(msg.Value)
	jobID, err := uuid.Parse(jobIDStr)
	if err != nil {
		log.Printf("Worker %d: Invalid job ID: %s", workerID, jobIDStr)
		// Commit invalid message to avoid reprocessing
		w.kafkaReader.CommitMessages(context.Background(), msg)
		return
	}

	log.Printf("Worker %d received job %s from partition %d", workerID, jobID, msg.Partition)

	// Fetch job from cache first (cache-aside pattern)
	job := w.cacheService.GetJob(jobID)

	if job == nil {
		// Cache miss - fetch from database
		log.Printf("Cache miss for job %s, fetching from database", jobID)
		job, err = w.jobRepository.FindByID(jobID)
		if err != nil {
			log.Printf("Worker %d: Job not found: %s", workerID, jobID)
			w.kafkaReader.CommitMessages(context.Background(), msg)
			return
		}

		// Cache for future requests
		w.cacheService.CacheJob(job)
	}

	// Process the job
	processErr := w.processJobInternal(job)

	if processErr != nil {
		log.Printf("Worker %d: Failed to process job %s: %v", workerID, jobID, processErr)

		// Handle failure with retry logic
		w.handleJobFailure(job, processErr)
	}

	// Acknowledge Kafka message (commit offset)
	// Only after successful DB update
	// Job will be retried via scheduler based on scheduledAt if it failed
	if err := w.kafkaReader.CommitMessages(context.Background(), msg); err != nil {
		log.Printf("Worker %d: Failed to commit message for job %s: %v", workerID, jobID, err)
		return
	}

	if processErr == nil {
		log.Printf("Worker %d: Job %s processed successfully and acknowledged", workerID, jobID)
	}
}

// processJobInternal processes the job based on its type.
//
// In a real system, this would:
// - PAYMENT_PROCESS: Call Stripe/PayPal API to charge card
// - EMAIL_CONFIRMATION: Call SendGrid/SES API to send email
//
// For this project, we simulate with time.Sleep to mimic API latency.
func (w *JobWorker) processJobInternal(job *model.Job) error {
	log.Printf("Processing job: id=%s, type=%s, clientId=%s, attempt=%d/%d",
		job.ID, job.Type, job.ClientID, job.Attempts+1, job.MaxRetries)

	// Simulate different processing times based on job type
	switch job.Type {
	case model.TypePaymentProcess:
		// Simulate Stripe API call (2 seconds)
		log.Printf("Simulating payment processing for job %s", job.ID)
		time.Sleep(2 * time.Second)
		log.Printf("Payment processed: %s", job.Payload)

	case model.TypeEmailConfirmation:
		// Simulate SendGrid API call (1 second)
		log.Printf("Simulating email send for job %s", job.ID)
		time.Sleep(1 * time.Second)
		log.Printf("Email sent: %s", job.Payload)

	default:
		return fmt.Errorf("unknown job type: %s", job.Type)
	}

	// Mark job as completed
	now := time.Now()
	job.Status = model.StatusCompleted
	job.CompletedAt = &now
	job.UpdatedAt = now

	if err := w.jobRepository.Save(job); err != nil {
		return fmt.Errorf("failed to save completed job: %w", err)
	}

	// Update cache with completed job
	w.cacheService.UpdateJob(job)

	log.Printf("Job %s completed successfully: type=%s, processingTime=%dms",
		job.ID, job.Type, getProcessingTime(job.Type))

	return nil
}

// handleJobFailure handles job failure with retry logic and exponential backoff.
//
// Retry Strategy:
// - Attempt 1 fails: Retry in 2^1 = 2 seconds
// - Attempt 2 fails: Retry in 2^2 = 4 seconds
// - Attempt 3 fails: Retry in 2^3 = 8 seconds
// - Attempt 4: Move to DEAD_LETTER (max 3 retries exceeded)
func (w *JobWorker) handleJobFailure(job *model.Job, jobErr error) {
	// Increment attempt counter
	job.Attempts++
	errMsg := jobErr.Error()
	job.ErrorMessage = &errMsg
	job.UpdatedAt = time.Now()

	if job.Attempts < job.MaxRetries {
		// Calculate exponential backoff delay: 2^attempts seconds
		delaySeconds := int64(math.Pow(2, float64(job.Attempts)))

		log.Printf("Job %s failed (attempt %d/%d), will retry in %ds: %s",
			job.ID, job.Attempts, job.MaxRetries, delaySeconds, jobErr.Error())

		// Set status back to PENDING for scheduler to pick up
		job.Status = model.StatusPending

		// Schedule for retry after exponential backoff delay
		retryAt := time.Now().Add(time.Duration(delaySeconds) * time.Second)
		job.ScheduledAt = &retryAt

	} else {
		// Max retries exceeded - move to dead letter queue
		log.Printf("Job %s moved to DEAD_LETTER after %d attempts: %s",
			job.ID, job.Attempts, jobErr.Error())

		job.Status = model.StatusDeadLetter
		now := time.Now()
		job.CompletedAt = &now
	}

	if err := w.jobRepository.Save(job); err != nil {
		log.Printf("Failed to save job failure state for %s: %v", job.ID, err)
	}

	// Update cache
	w.cacheService.UpdateJob(job)
}

// getProcessingTime returns the simulated processing time for a job type.
func getProcessingTime(jobType model.JobType) int {
	switch jobType {
	case model.TypePaymentProcess:
		return 2000
	case model.TypeEmailConfirmation:
		return 1000
	default:
		return 0
	}
}