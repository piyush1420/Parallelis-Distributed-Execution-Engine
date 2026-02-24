package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"distributed-job-processor/model"
)

// CacheService provides caching for job details using Redis.
//
// Strategy: Cache-Aside Pattern
// 1. Worker receives job ID from Kafka
// 2. Check Redis cache for job details
// 3. Cache hit: Use cached data (saves DB query)
// 4. Cache miss: Query database, then cache result
//
// Benefits:
// - Reduces database load during flash sales
// - Faster job processing (Redis is faster than PostgreSQL)
// - Protects database from overload during 100x traffic spikes
//
// Redis Key Format: job:{jobId}
// Redis Value: Serialized Job object (JSON)
// TTL: 15 minutes (configurable)
//
// Example Performance:
// - Without cache: 10ms DB query per job
// - With cache (80% hit rate): 2ms average (0.8 * 1ms + 0.2 * 10ms)
// - At 1000 jobs/min: Saves 8000ms = 8 seconds of DB time
type CacheService struct {
	redisClient      *redis.Client
	jobCacheTTLMinutes int
}

var ctx = context.Background()

// NewCacheService creates a new CacheService with the given Redis client.
func NewCacheService(redisClient *redis.Client) *CacheService {
	ttl := 15 // default
	if val := os.Getenv("CACHE_JOB_TTL_MINUTES"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			ttl = parsed
		}
	}
	return &CacheService{
		redisClient:      redisClient,
		jobCacheTTLMinutes: ttl,
	}
}

// GetJob retrieves a job from cache.
// Returns the Job if found in cache, nil otherwise.
func (cs *CacheService) GetJob(jobID uuid.UUID) *model.Job {
	key := cs.getJobCacheKey(jobID)

	data, err := cs.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		if err == redis.Nil {
			log.Printf("Cache MISS for job: %s", jobID)
		} else {
			log.Printf("Error getting job %s from cache: %v", jobID, err)
		}
		return nil
	}

	var job model.Job
	if err := json.Unmarshal(data, &job); err != nil {
		log.Printf("Error deserializing job %s from cache: %v", jobID, err)
		return nil
	}

	log.Printf("Cache HIT for job: %s", jobID)
	return &job
}

// CacheJob stores a job in the cache.
func (cs *CacheService) CacheJob(job *model.Job) {
	if job == nil || job.ID == uuid.Nil {
		return
	}

	key := cs.getJobCacheKey(job.ID)
	ttl := time.Duration(cs.jobCacheTTLMinutes) * time.Minute

	data, err := json.Marshal(job)
	if err != nil {
		log.Printf("Error serializing job %s for cache: %v", job.ID, err)
		return
	}

	if err := cs.redisClient.Set(ctx, key, data, ttl).Err(); err != nil {
		log.Printf("Error caching job %s: %v", job.ID, err)
		return
	}

	log.Printf("Cached job: %s (TTL: %d minutes)", job.ID, cs.jobCacheTTLMinutes)
}

// InvalidateJob deletes a job from cache.
// Call this when job is updated to keep cache consistent.
func (cs *CacheService) InvalidateJob(jobID uuid.UUID) {
	key := cs.getJobCacheKey(jobID)

	if err := cs.redisClient.Del(ctx, key).Err(); err != nil {
		log.Printf("Error invalidating job %s: %v", jobID, err)
		return
	}

	log.Printf("Invalidated cache for job: %s", jobID)
}

// UpdateJob updates a job in cache after modification.
func (cs *CacheService) UpdateJob(job *model.Job) {
	cs.InvalidateJob(job.ID)
	cs.CacheJob(job)
}

// GetCacheInfo returns cache statistics for monitoring.
func (cs *CacheService) GetCacheInfo() string {
	keys, err := cs.redisClient.Keys(ctx, "job:*").Result()
	if err != nil {
		log.Printf("Error getting cache info: %v", err)
		return "Cache info unavailable"
	}
	return fmt.Sprintf("Cached jobs: %d", len(keys))
}

// ClearAllJobCaches clears all job caches (admin function).
func (cs *CacheService) ClearAllJobCaches() {
	keys, err := cs.redisClient.Keys(ctx, "job:*").Result()
	if err != nil {
		log.Printf("Error clearing job caches: %v", err)
		return
	}

	if len(keys) > 0 {
		if err := cs.redisClient.Del(ctx, keys...).Err(); err != nil {
			log.Printf("Error clearing job caches: %v", err)
			return
		}
	}

	log.Println("Cleared all job caches")
}

// getJobCacheKey returns the Redis key for job caching.
func (cs *CacheService) getJobCacheKey(jobID uuid.UUID) string {
	return "job:" + jobID.String()
}