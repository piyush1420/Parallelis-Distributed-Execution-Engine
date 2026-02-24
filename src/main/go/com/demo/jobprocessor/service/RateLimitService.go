package service

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RateLimitService provides rate limiting using Redis and token bucket algorithm.
//
// Strategy: Token Bucket
// - Each client gets a bucket with MAX_REQUESTS tokens
// - Each request consumes 1 token
// - Bucket refills to MAX_REQUESTS every WINDOW_SECONDS
//
// Example: 100 requests per 60 seconds
// - Client can burst up to 100 requests immediately
// - Then must wait for bucket to refill
//
// Redis Key Format: rate_limit:{clientId}
// Redis Value: Hash with {count: Integer, resetTime: Long}
//
// Benefits:
// - Prevents one bot from monopolizing system during flash sales
// - Ensures fair access to limited inventory
// - Protects backend services from overload
type RateLimitService struct {
	redisClient   *redis.Client
	enabled       bool
	maxRequests   int
	windowSeconds int
}

// NewRateLimitService creates a new RateLimitService with the given Redis client.
func NewRateLimitService(redisClient *redis.Client) *RateLimitService {
	enabled := true
	if val := os.Getenv("RATE_LIMIT_ENABLED"); val == "false" {
		enabled = false
	}

	maxRequests := 100
	if val := os.Getenv("RATE_LIMIT_MAX_REQUESTS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			maxRequests = parsed
		}
	}

	windowSeconds := 60
	if val := os.Getenv("RATE_LIMIT_WINDOW_SECONDS"); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil {
			windowSeconds = parsed
		}
	}

	return &RateLimitService{
		redisClient:   redisClient,
		enabled:       enabled,
		maxRequests:   maxRequests,
		windowSeconds: windowSeconds,
	}
}

// IsAllowed checks if the client is allowed to make a request.
// Returns true if allowed, false if rate limit exceeded.
func (s *RateLimitService) IsAllowed(clientID string) bool {
	if !s.enabled {
		return true
	}

	key := s.getRateLimitKey(clientID)
	now := time.Now().Unix()

	// Get current count and reset time from Redis
	count, errCount := s.redisClient.HGet(ctx, key, "count").Int()
	resetTime, errReset := s.redisClient.HGet(ctx, key, "resetTime").Int64()

	// First request or bucket has been reset
	if errCount != nil || errReset != nil || now >= resetTime {
		// Initialize new bucket
		pipe := s.redisClient.Pipeline()
		pipe.HSet(ctx, key, "count", 1)
		pipe.HSet(ctx, key, "resetTime", now+int64(s.windowSeconds))
		pipe.Expire(ctx, key, time.Duration(s.windowSeconds+10)*time.Second) // Extra 10s buffer
		if _, err := pipe.Exec(ctx); err != nil {
			log.Printf("Error initializing rate limit for client %s: %v", clientID, err)
			// Fail open: Allow request if Redis is down
			return true
		}

		log.Printf("Rate limit initialized for client %s: 1/%d requests", clientID, s.maxRequests)
		return true
	}

	// Check if under limit
	if count < s.maxRequests {
		// Increment counter
		if err := s.redisClient.HIncrBy(ctx, key, "count", 1).Err(); err != nil {
			log.Printf("Error incrementing rate limit for client %s: %v", clientID, err)
			return true
		}
		log.Printf("Rate limit for client %s: %d/%d requests", clientID, count+1, s.maxRequests)
		return true
	}

	// Rate limit exceeded
	secondsUntilReset := resetTime - now
	log.Printf("Rate limit exceeded for client %s: %d/%d requests, resets in %ds",
		clientID, count, s.maxRequests, secondsUntilReset)
	return false
}

// GetRemainingRequests returns the number of remaining requests for a client in the current window.
func (s *RateLimitService) GetRemainingRequests(clientID string) int64 {
	if !s.enabled {
		return int64(s.maxRequests)
	}

	key := s.getRateLimitKey(clientID)
	now := time.Now().Unix()

	count, errCount := s.redisClient.HGet(ctx, key, "count").Int()
	resetTime, errReset := s.redisClient.HGet(ctx, key, "resetTime").Int64()

	if errCount != nil || errReset != nil || now >= resetTime {
		return int64(s.maxRequests)
	}

	remaining := s.maxRequests - count
	if remaining < 0 {
		remaining = 0
	}
	return int64(remaining)
}

// GetSecondsUntilReset returns seconds until rate limit resets for a client.
// Returns 0 if no active limit.
func (s *RateLimitService) GetSecondsUntilReset(clientID string) int64 {
	if !s.enabled {
		return 0
	}

	key := s.getRateLimitKey(clientID)
	now := time.Now().Unix()

	resetTime, err := s.redisClient.HGet(ctx, key, "resetTime").Int64()
	if err != nil || now >= resetTime {
		return 0
	}

	return resetTime - now
}

// ResetRateLimit resets the rate limit for a client (admin function).
func (s *RateLimitService) ResetRateLimit(clientID string) {
	key := s.getRateLimitKey(clientID)
	if err := s.redisClient.Del(ctx, key).Err(); err != nil {
		log.Printf("Error resetting rate limit for client %s: %v", clientID, err)
		return
	}
	log.Printf("Rate limit reset for client: %s", clientID)
}

// getRateLimitKey returns the Redis key for rate limiting.
func (s *RateLimitService) getRateLimitKey(clientID string) string {
	return "rate_limit:" + clientID
}