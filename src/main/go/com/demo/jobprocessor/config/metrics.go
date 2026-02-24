package config

import (
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
)

// Metrics provides lightweight application metrics for monitoring.
// Exposes counters and histograms at GET /metrics for Prometheus scraping
// or manual inspection.
//
// Tracked metrics:
// - HTTP request count and latency (by endpoint, method, status)
// - Job processing count (by type, status)
// - Kafka message count (produced, consumed, failed)
// - Redis cache hit/miss ratio
// - Rate limit rejections per client

type Metrics struct {
	// HTTP metrics
	httpRequestsTotal   map[string]*atomic.Int64
	httpLatencySum      map[string]*atomic.Int64
	httpLatencyCount    map[string]*atomic.Int64
	httpMu              sync.RWMutex

	// Job metrics
	jobsCreated         atomic.Int64
	jobsCompleted       atomic.Int64
	jobsFailed          atomic.Int64
	jobsDeadLettered    atomic.Int64
	jobsRetried         atomic.Int64

	// Kafka metrics
	kafkaMessagesProduced atomic.Int64
	kafkaMessagesConsumed atomic.Int64
	kafkaProduceErrors    atomic.Int64

	// Redis metrics
	cacheHits           atomic.Int64
	cacheMisses         atomic.Int64
	rateLimitRejections atomic.Int64

	// Worker metrics
	activeWorkers       atomic.Int64
	processingTimeSum   atomic.Int64
	processingTimeCount atomic.Int64
}

// Global metrics instance
var appMetrics = &Metrics{
	httpRequestsTotal: make(map[string]*atomic.Int64),
	httpLatencySum:    make(map[string]*atomic.Int64),
	httpLatencyCount:  make(map[string]*atomic.Int64),
}

// GetMetrics returns the global metrics instance.
func GetMetrics() *Metrics {
	return appMetrics
}

// RecordHTTPRequest records an HTTP request metric.
func (m *Metrics) RecordHTTPRequest(method, path string, status int, duration time.Duration) {
	key := method + " " + path + " " + strconv.Itoa(status)

	m.httpMu.Lock()
	if _, ok := m.httpRequestsTotal[key]; !ok {
		m.httpRequestsTotal[key] = &atomic.Int64{}
		m.httpLatencySum[key] = &atomic.Int64{}
		m.httpLatencyCount[key] = &atomic.Int64{}
	}
	m.httpMu.Unlock()

	m.httpMu.RLock()
	m.httpRequestsTotal[key].Add(1)
	m.httpLatencySum[key].Add(duration.Microseconds())
	m.httpLatencyCount[key].Add(1)
	m.httpMu.RUnlock()
}

// Job metric helpers
func (m *Metrics) IncJobsCreated()      { m.jobsCreated.Add(1) }
func (m *Metrics) IncJobsCompleted()    { m.jobsCompleted.Add(1) }
func (m *Metrics) IncJobsFailed()       { m.jobsFailed.Add(1) }
func (m *Metrics) IncJobsDeadLettered() { m.jobsDeadLettered.Add(1) }
func (m *Metrics) IncJobsRetried()      { m.jobsRetried.Add(1) }

// Kafka metric helpers
func (m *Metrics) IncKafkaProduced()     { m.kafkaMessagesProduced.Add(1) }
func (m *Metrics) IncKafkaConsumed()     { m.kafkaMessagesConsumed.Add(1) }
func (m *Metrics) IncKafkaProduceError() { m.kafkaProduceErrors.Add(1) }

// Cache metric helpers
func (m *Metrics) IncCacheHit()             { m.cacheHits.Add(1) }
func (m *Metrics) IncCacheMiss()            { m.cacheMisses.Add(1) }
func (m *Metrics) IncRateLimitRejection()   { m.rateLimitRejections.Add(1) }

// Worker metric helpers
func (m *Metrics) IncActiveWorkers()  { m.activeWorkers.Add(1) }
func (m *Metrics) DecActiveWorkers()  { m.activeWorkers.Add(-1) }
func (m *Metrics) RecordProcessingTime(d time.Duration) {
	m.processingTimeSum.Add(d.Microseconds())
	m.processingTimeCount.Add(1)
}

// MetricsMiddleware records HTTP request metrics for every request.
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start)
		GetMetrics().RecordHTTPRequest(c.Request.Method, c.FullPath(), c.Writer.Status(), duration)
	}
}

// MetricsHandler returns current metrics as JSON.
// GET /metrics
func MetricsHandler(c *gin.Context) {
	m := GetMetrics()

	// Calculate cache hit ratio
	hits := m.cacheHits.Load()
	misses := m.cacheMisses.Load()
	hitRatio := float64(0)
	if hits+misses > 0 {
		hitRatio = float64(hits) / float64(hits+misses) * 100
	}

	// Calculate average processing time
	avgProcessing := float64(0)
	if m.processingTimeCount.Load() > 0 {
		avgProcessing = float64(m.processingTimeSum.Load()) / float64(m.processingTimeCount.Load()) / 1000 // ms
	}

	// Build HTTP endpoint metrics
	httpMetrics := make(map[string]map[string]interface{})
	m.httpMu.RLock()
	for key, count := range m.httpRequestsTotal {
		avgLatency := float64(0)
		if m.httpLatencyCount[key].Load() > 0 {
			avgLatency = float64(m.httpLatencySum[key].Load()) / float64(m.httpLatencyCount[key].Load()) / 1000 // ms
		}
		httpMetrics[key] = map[string]interface{}{
			"count":          count.Load(),
			"avg_latency_ms": avgLatency,
		}
	}
	m.httpMu.RUnlock()

	c.JSON(200, gin.H{
		"jobs": gin.H{
			"created":       m.jobsCreated.Load(),
			"completed":     m.jobsCompleted.Load(),
			"failed":        m.jobsFailed.Load(),
			"dead_lettered": m.jobsDeadLettered.Load(),
			"retried":       m.jobsRetried.Load(),
		},
		"kafka": gin.H{
			"messages_produced": m.kafkaMessagesProduced.Load(),
			"messages_consumed": m.kafkaMessagesConsumed.Load(),
			"produce_errors":    m.kafkaProduceErrors.Load(),
		},
		"cache": gin.H{
			"hits":      hits,
			"misses":    misses,
			"hit_ratio": hitRatio,
		},
		"rate_limiting": gin.H{
			"rejections": m.rateLimitRejections.Load(),
		},
		"workers": gin.H{
			"active":                m.activeWorkers.Load(),
			"avg_processing_time_ms": avgProcessing,
		},
		"http_endpoints": httpMetrics,
	})
}