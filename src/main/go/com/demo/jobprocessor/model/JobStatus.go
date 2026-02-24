package model

// JobStatus represents the lifecycle status of a job.
type JobStatus string

const (
	// StatusPending - Job has been created and is waiting to be scheduled
	StatusPending JobStatus = "PENDING"

	// StatusRunning - Job has been picked up by scheduler and sent to Kafka
	StatusRunning JobStatus = "RUNNING"

	// StatusCompleted - Job is completed
	StatusCompleted JobStatus = "COMPLETED"

	// StatusFailed - Job has failed
	StatusFailed JobStatus = "FAILED"

	// StatusDeadLetter - Job has exceeded max retries and moved to dead letter
	StatusDeadLetter JobStatus = "DEAD_LETTER"
)