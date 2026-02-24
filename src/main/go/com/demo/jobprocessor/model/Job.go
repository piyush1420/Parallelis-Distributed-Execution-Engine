package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Job represents an order processing job in the distributed e-commerce system.
//
// Context: Flash Sale / Normal Sale Order Processing
// - During flash sales (Black Friday), system receives thousands of orders per minute
// - Each order generates multiple jobs: PAYMENT_PROCESS → INVENTORY_UPDATE → EMAIL_CONFIRMATION
// - Jobs are persisted in PostgreSQL (source of truth) and cached in Redis for performance
//
// Reliability Guarantees:
// - Zero job loss during worker failures (Kafka consumer acknowledgment after DB commit)
// - Automatic retry with exponential backoff for transient failures
// - Dead letter queue for permanently failed jobs after max retries
type Job struct {
	// Unique identifier for the job (UUID)
	ID uuid.UUID `json:"id" gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`

	// Client identifier for rate limiting and tracking
	ClientID string `json:"clientId" gorm:"column:client_id;not null;size:100;index:idx_client_id"`

	// Type of job to be processed
	Type JobType `json:"type" gorm:"column:type;not null;size:50"`

	// Current status of the job in its lifecycle
	Status JobStatus `json:"status" gorm:"column:status;not null;size:20;index:idx_status_scheduled_at"`

	// Job payload containing the data to be processed
	Payload string `json:"payload" gorm:"column:payload;not null;type:text"`

	// Number of times this job has been attempted
	Attempts int `json:"attempts" gorm:"column:attempts;not null;default:0"`

	// Maximum number of retry attempts before moving to DEAD_LETTER
	MaxRetries int `json:"maxRetries" gorm:"column:max_retries;not null;default:3"`

	// Timestamp when the job was created
	CreatedAt time.Time `json:"createdAt" gorm:"column:created_at;not null;autoCreateTime;index:idx_created_at"`

	// Timestamp when the job should be/was scheduled for processing
	ScheduledAt *time.Time `json:"scheduledAt,omitempty" gorm:"column:scheduled_at;not null;index:idx_status_scheduled_at"`

	// Timestamp when the job completed (successfully or failed permanently)
	CompletedAt *time.Time `json:"completedAt,omitempty" gorm:"column:completed_at"`

	// Optional error message if job failed
	ErrorMessage *string `json:"errorMessage,omitempty" gorm:"column:error_message;type:text"`

	// Timestamp when the job was last updated
	UpdatedAt time.Time `json:"updatedAt" gorm:"column:updated_at;autoUpdateTime"`
}

// TableName specifies the database table name for the Job model.
func (Job) TableName() string {
	return "jobs"
}

// BeforeCreate is a GORM hook that runs before inserting a new record.
// Sets UUID and default values.
func (j *Job) BeforeCreate(tx *gorm.DB) error {
	if j.ID == uuid.Nil {
		j.ID = uuid.New()
	}
	if j.ScheduledAt == nil {
		now := time.Now()
		j.ScheduledAt = &now
	}
	if j.Attempts == 0 {
		j.Attempts = 0
	}
	if j.MaxRetries == 0 {
		j.MaxRetries = 3
	}
	return nil
}

// BeforeUpdate is a GORM hook that runs before updating a record.
// Sets scheduledAt if not explicitly set.
func (j *Job) BeforeUpdate(tx *gorm.DB) error {
	if j.ScheduledAt == nil {
		now := time.Now()
		j.ScheduledAt = &now
	}
	return nil
}

// NewJob creates a new Job with default values.
func NewJob(clientID string, jobType JobType, payload string) *Job {
	now := time.Now()
	return &Job{
		ID:         uuid.New(),
		ClientID:   clientID,
		Type:       jobType,
		Status:     StatusPending,
		Payload:    payload,
		Attempts:   0,
		MaxRetries: 3,
		CreatedAt:  now,
		ScheduledAt: &now,
	}
}