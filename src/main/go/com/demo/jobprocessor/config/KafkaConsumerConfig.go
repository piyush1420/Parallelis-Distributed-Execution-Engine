package config

import (
	"os"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaConsumerConfig configures Kafka consumer for worker instances.
//
// Workers consume job IDs from the Kafka topic and process them.
//
// Configuration:
// - Consumer group: "job-workers" (enables parallel processing)
// - Manual acknowledgment: Only commit after successful DB update
// - auto-offset-reset=earliest: Start from beginning if no offset
//
// Multiple workers can run in parallel, each consuming from different partitions.

// GetBootstrapServers returns the Kafka bootstrap servers from env or default.
func GetBootstrapServers() string {
	servers := os.Getenv("KAFKA_BOOTSTRAP_SERVERS")
	if servers == "" {
		return "localhost:9092"
	}
	return servers
}

// GetConsumerGroupID returns the consumer group ID from env or default.
func GetConsumerGroupID() string {
	groupID := os.Getenv("KAFKA_CONSUMER_GROUP_ID")
	if groupID == "" {
		return "job-workers"
	}
	return groupID
}

// NewKafkaConsumerReader creates a configured Kafka reader (consumer) for reliable message processing.
//
// Configuration mirrors the Java version:
// - Consumer group for parallel processing
// - Start from earliest offset if no offset exists (don't lose jobs)
// - Manual commit: commit only after successful processing
// - Fetch configuration for better throughput
// - Session timeout and heartbeat settings
func NewKafkaConsumerReader(topic string) *kafka.Reader {
	return kafka.NewReader(kafka.ReaderConfig{
			Brokers: []string{GetBootstrapServers()},
			Topic:   topic,
			GroupID: GetConsumerGroupID(),

			// Start from earliest if no offset exists (don't lose jobs)
			StartOffset: kafka.FirstOffset,

			// Fetch configuration for better throughput
			MinBytes: 1,
			MaxWait:  500 * time.Millisecond,

			// Session timeout and heartbeat
			SessionTimeout: 30 * time.Second,
			HeartbeatInterval: 10 * time.Second,

			// Concurrency: Number of concurrent consumers per reader
			// Can be scaled up to match number of Kafka partitions (16)
			MaxAttempts: 3,
		},
		// Manual commit: We'll commit manually after successful processing
		// In kafka-go, this is done by NOT enabling auto-commit
		// and calling reader.CommitMessages() explicitly
	)
}

// CommitMessage manually commits the offset after successful processing.
//
// Manual acknowledgment ensures we only commit the offset after:
// 1. Successfully processing the job
// 2. Successfully updating the database
//
// This guarantees at-least-once delivery semantics.
func CommitMessage(reader *kafka.Reader, msg kafka.Message) error {
	return reader.CommitMessages(nil, msg)
}