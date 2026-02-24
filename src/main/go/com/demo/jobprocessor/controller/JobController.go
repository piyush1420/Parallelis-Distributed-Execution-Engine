package controller

import (
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"distributed-job-processor/dto"
	"distributed-job-processor/model"
	"distributed-job-processor/service"
)

// JobController handles REST API endpoints for job management.
//
// Endpoints:
// - POST /api/jobs - Create a new job (returns 202 Accepted)
// - GET /api/jobs/:id - Get job status by ID
// - GET /api/jobs?clientId={id} - Get all jobs for a client
// - GET /api/jobs/stats - Get system statistics
//
// Features:
// - Rate limiting: 100 requests/minute per client (via Redis)
// - Input validation
// - Error handling
type JobController struct {
	jobService      *service.JobService
	rateLimitService *service.RateLimitService
}

// NewJobController creates a new JobController with the given services.
func NewJobController(jobService *service.JobService, rateLimitService *service.RateLimitService) *JobController {
	return &JobController{
		jobService:      jobService,
		rateLimitService: rateLimitService,
	}
}

// RegisterRoutes registers all job-related routes with the Gin router.
func (jc *JobController) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("", jc.CreateJob)
	r.GET("/stats", jc.GetStats)
	r.GET("/health", jc.Health)
	r.GET("/:id", jc.GetJob)
	r.GET("", jc.GetJobsByClient)
}

// CreateJob creates a new order processing job.
//
// The job is created in PENDING status and will be picked up by the scheduler
// for processing. Returns 202 Accepted to indicate the request has been accepted
// but processing is asynchronous.
//
// Rate Limiting:
// - 100 requests per minute per client
// - Enforced via Redis token bucket
// - Returns 429 Too Many Requests if exceeded
//
// Example request:
// POST /api/jobs
// Headers: X-Client-Id: customer-12345
// Body: {
//   "type": "PAYMENT_PROCESS",
//   "payload": "order_ORD123|user@email.com|$99.99"
// }
func (jc *JobController) CreateJob(c *gin.Context) {
	clientID := c.GetHeader("X-Client-Id")
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Client-Id header is required"})
		return
	}

	var request dto.JobRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid input", "details": err.Error()})
		return
	}

	log.Printf("Received job creation request: clientId=%s, type=%s", clientID, request.Type)

	// Rate limiting check
	if !jc.rateLimitService.IsAllowed(clientID) {
		remaining := jc.rateLimitService.GetRemainingRequests(clientID)
		log.Printf("Rate limit exceeded for client: %s, remaining: %d", clientID, remaining)

		c.Header("X-RateLimit-Limit", "100")
		c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
		c.JSON(http.StatusTooManyRequests, nil)
		return
	}

	job, err := jc.jobService.CreateJob(clientID, &request)
	if err != nil {
		log.Printf("Failed to create job: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create job"})
		return
	}

	response := dto.JobResponseFrom(job)
	remaining := jc.rateLimitService.GetRemainingRequests(clientID)

	log.Printf("Job created: jobId=%s, status=%s, remaining requests: %d",
		job.ID, job.Status, remaining)

	c.Header("X-RateLimit-Limit", "100")
	c.Header("X-RateLimit-Remaining", strconv.FormatInt(remaining, 10))
	c.JSON(http.StatusAccepted, response)
}

// GetJob gets job status by ID.
//
// Returns the current status and details of a job. Clients can poll this
// endpoint to check if their order has been processed.
//
// Example request:
// GET /api/jobs/550e8400-e29b-41d4-a716-446655440000
func (jc *JobController) GetJob(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid job ID format"})
		return
	}

	log.Printf("Retrieving job: %s", id)

	job, err := jc.jobService.GetJob(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Job not found", "id": id.String()})
		return
	}

	response := dto.JobResponseFrom(job)
	c.JSON(http.StatusOK, response)
}

// GetJobsByClient gets all jobs for a specific client.
//
// Useful for client-specific dashboards and order history.
//
// Example request:
// GET /api/jobs?clientId=customer-12345
func (jc *JobController) GetJobsByClient(c *gin.Context) {
	clientID := c.Query("clientId")
	if clientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "clientId query parameter is required"})
		return
	}

	log.Printf("Retrieving jobs for client: %s", clientID)

	jobs, err := jc.jobService.GetJobsByClient(clientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve jobs"})
		return
	}

	var responses []dto.JobResponse
	for _, job := range jobs {
		responses = append(responses, dto.JobResponseFrom(&job))
	}

	if responses == nil {
		responses = []dto.JobResponse{}
	}

	c.JSON(http.StatusOK, responses)
}

// GetStats returns system statistics.
//
// Returns count of jobs by status, useful for monitoring dashboards.
//
// Example response:
// {
//   "PENDING": 150,
//   "RUNNING": 25,
//   "COMPLETED": 10450,
//   "FAILED": 5,
//   "DEAD_LETTER": 2
// }
func (jc *JobController) GetStats(c *gin.Context) {
	log.Println("Retrieving system statistics")

	stats := map[string]int64{
		"PENDING":     jc.jobService.CountJobsByStatus(model.StatusPending),
		"RUNNING":     jc.jobService.CountJobsByStatus(model.StatusRunning),
		"COMPLETED":   jc.jobService.CountJobsByStatus(model.StatusCompleted),
		"FAILED":      jc.jobService.CountJobsByStatus(model.StatusFailed),
		"DEAD_LETTER": jc.jobService.CountJobsByStatus(model.StatusDeadLetter),
	}

	c.JSON(http.StatusOK, stats)
}

// Health check endpoint.
func (jc *JobController) Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":  "UP",
		"service": "job-processor-api",
	})
}