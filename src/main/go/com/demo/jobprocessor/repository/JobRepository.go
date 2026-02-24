package repository

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"distributed-job-processor/model"
)

// JobRepository provides persistence operations for the Job entity.
// Equivalent to Spring Data JPA's JpaRepository with custom queries.
type JobRepository struct {
	db *gorm.DB
}

// NewJobRepository creates a new JobRepository with the given database connection.
func NewJobRepository(db *gorm.DB) *JobRepository {
	return &JobRepository{db: db}
}

// Save creates or updates a job in the database.
func (r *JobRepository) Save(job *model.Job) error {
	return r.db.Save(job).Error
}

// FindByID finds a job by its UUID.
func (r *JobRepository) FindByID(id uuid.UUID) (*model.Job, error) {
	var job model.Job
	err := r.db.First(&job, "id = ?", id).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

// FindAll returns all jobs.
func (r *JobRepository) FindAll() ([]model.Job, error) {
	var jobs []model.Job
	err := r.db.Find(&jobs).Error
	return jobs, err
}

// Delete removes a job from the database.
func (r *JobRepository) Delete(job *model.Job) error {
	return r.db.Delete(job).Error
}

// FindByStatusAndScheduledAtBefore finds all jobs with a specific status
// that are scheduled to run before the given time.
// This is the primary query used by the scheduler to find jobs ready for processing.
//
// Equivalent to:
// SELECT j FROM Job j WHERE j.status = :status AND j.scheduledAt <= :scheduledAt ORDER BY j.scheduledAt ASC
func (r *JobRepository) FindByStatusAndScheduledAtBefore(status model.JobStatus, scheduledAt time.Time) ([]model.Job, error) {
	var jobs []model.Job
	err := r.db.Where("status = ? AND scheduled_at <= ?", status, scheduledAt).
		Order("scheduled_at ASC").
		Find(&jobs).Error
	return jobs, err
}

// FindByClientID finds all jobs by client ID (useful for tracking and analytics).
func (r *JobRepository) FindByClientID(clientID string) ([]model.Job, error) {
	var jobs []model.Job
	err := r.db.Where("client_id = ?", clientID).Find(&jobs).Error
	return jobs, err
}

// FindByStatus finds all jobs by status.
func (r *JobRepository) FindByStatus(status model.JobStatus) ([]model.Job, error) {
	var jobs []model.Job
	err := r.db.Where("status = ?", status).Find(&jobs).Error
	return jobs, err
}

// CountByStatus counts jobs by status (useful for monitoring and dashboards).
func (r *JobRepository) CountByStatus(status model.JobStatus) (int64, error) {
	var count int64
	err := r.db.Model(&model.Job{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

// FindStuckJobs finds jobs that have been running for longer than expected (potential stuck jobs).
//
// Equivalent to:
// SELECT j FROM Job j WHERE j.status = :status AND j.updatedAt < :updatedBefore
func (r *JobRepository) FindStuckJobs(status model.JobStatus, updatedBefore time.Time) ([]model.Job, error) {
	var jobs []model.Job
	err := r.db.Where("status = ? AND updated_at < ?", status, updatedBefore).
		Find(&jobs).Error
	return jobs, err
}