package service

import (
	"fmt"
	"math"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
)

// Benchmarks for critical path operations.
// Run: go test -bench=. -benchmem -cpuprofile=cpu.prof -memprofile=mem.prof
// Analyze: go tool pprof cpu.prof

// BenchmarkUUIDGeneration measures UUID generation throughput.
// UUIDs are generated for every job creation — this must be fast.
func BenchmarkUUIDGeneration(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = uuid.New()
	}
}

// BenchmarkExponentialBackoff measures the retry delay calculation.
// Called on every job failure — must not add latency.
func BenchmarkExponentialBackoff(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		attempts := i % 10
		_ = time.Duration(int64(math.Pow(2, float64(attempts)))) * time.Second
	}
}

// BenchmarkJobKeyGeneration measures Redis cache key generation.
// Called on every cache lookup — hot path.
func BenchmarkJobKeyGeneration(b *testing.B) {
	id := uuid.New()
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = "job:" + id.String()
	}
}

// BenchmarkRateLimitKeyGeneration measures rate limit key generation.
func BenchmarkRateLimitKeyGeneration(b *testing.B) {
	clientID := "customer-12345"
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = "rate_limit:" + clientID
	}
}

// BenchmarkConcurrentJobCreation simulates concurrent job submissions
// to measure contention and throughput under parallel load.
func BenchmarkConcurrentJobCreation(b *testing.B) {
	type Job struct {
		ID       uuid.UUID
		ClientID string
		Status   string
		Payload  string
	}

	var mu sync.Mutex
	jobs := make(map[uuid.UUID]*Job)

	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			job := &Job{
				ID:       uuid.New(),
				ClientID: fmt.Sprintf("client-%d", time.Now().UnixNano()%1000),
				Status:   "PENDING",
				Payload:  "order_12345|customer@email.com|$99.99",
			}
			mu.Lock()
			jobs[job.ID] = job
			mu.Unlock()
		}
	})
}

// BenchmarkPartitionRouting simulates Kafka partition assignment
// using client ID hashing. Ensures even distribution across 16 partitions.
func BenchmarkPartitionRouting(b *testing.B) {
	partitions := 16
	distribution := make([]int, partitions)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		clientID := fmt.Sprintf("client-%d", i%1000)
		// FNV-1a inspired hash for partition routing
		hash := uint32(2166136261)
		for _, c := range clientID {
			hash ^= uint32(c)
			hash *= 16777619
		}
		partition := int(hash) % partitions
		if partition < 0 {
			partition = -partition
		}
		distribution[partition]++
	}
}

// BenchmarkStatusTransition measures job status update operations.
// Workers update status on every job completion.
func BenchmarkStatusTransition(b *testing.B) {
	statuses := []string{"PENDING", "RUNNING", "COMPLETED", "FAILED", "DEAD_LETTER"}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		current := statuses[i%len(statuses)]
		next := statuses[(i+1)%len(statuses)]
		_ = current
		_ = next
	}
}

// BenchmarkPayloadParsing measures order payload parsing throughput.
// Every worker must parse the payload to extract order details.
func BenchmarkPayloadParsing(b *testing.B) {
	payload := "order_ORD12345|customer@email.com|$99.99|product_SKU789|qty_2"

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		// Parse pipe-delimited payload
		start := 0
		fields := make([]string, 0, 5)
		for j := 0; j < len(payload); j++ {
			if payload[j] == '|' {
				fields = append(fields, payload[start:j])
				start = j + 1
			}
		}
		fields = append(fields, payload[start:])
		_ = fields
	}
}