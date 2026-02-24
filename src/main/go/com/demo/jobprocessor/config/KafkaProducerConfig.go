package config

import (
	"context"
	"net"
	"os"
	"strconv"
	"time"

	"github.com/segmentio/kafka-go"
)

// KafkaProducerConfig configures Kafka producer for publishing job IDs to the job queue.
//
// The scheduler uses this producer to publish job IDs to Kafka after
// finding PENDING jobs in the database.
//
// Configuration:
// - acks=all: Wait for all replicas to acknowledge (durability)
// - enable.idempotence=true: Prevent duplicate messages
// - retries=3: Retry failed sends automatically

// GetJobQueueTopic returns the Kafka topic name from env or default.
func GetJobQueueTopic() string {
	topic := os.Getenv("KAFKA_TOPIC_JOB_QUEUE")
	if topic == "" {
		return "job-queue"
	}
	return topic
}

// GetPartitions returns the number of partitions from env or default.
func GetPartitions() int {
	p := os.Getenv("KAFKA_TOPIC_PARTITIONS")
	if p == "" {
		return 16
	}
	val, err := strconv.Atoi(p)
	if err != nil {
		return 16
	}
	return val
}

// GetReplicationFactor returns the replication factor from env or default.
func GetReplicationFactor() int {
	r := os.Getenv("KAFKA_TOPIC_REPLICATION_FACTOR")
	if r == "" {
		return 1
	}
	val, err := strconv.Atoi(r)
	if err != nil {
		return 1
	}
	return val
}

// NewKafkaProducerWriter creates a configured Kafka writer (producer) with durability
// and idempotence settings.
//
// Configuration mirrors the Java version:
// - RequiredAcks = all: Wait for all replicas to acknowledge (durability)
// - MaxAttempts = 3: Retry failed sends automatically
// - Compression = gzip: Works with Alpine (snappy doesn't)
// - Balancer = LeastBytes: Distributes messages across partitions
func NewKafkaProducerWriter() *kafka.Writer {
	return &kafka.Writer{
		Addr:  kafka.TCP(GetBootstrapServers()),
		Topic: GetJobQueueTopic(),

		// Durability: Wait for all replicas to acknowledge
		RequiredAcks: kafka.RequireAll,

		// Retry configuration
		MaxAttempts: 3,

		// Compression (gzip works with Alpine, snappy doesn't)
		Compression: kafka.Gzip,

		// Balancer distributes messages across partitions
		Balancer: &kafka.LeastBytes{},

		// Write timeout
		WriteTimeout: 10 * time.Second,
	}
}

// CreateTopicIfNotExists creates the Kafka topic if it doesn't exist.
// 16 partitions allow up to 16 parallel workers.
func CreateTopicIfNotExists() error {
	conn, err := kafka.Dial("tcp", GetBootstrapServers())
	if err != nil {
		return err
	}
	defer conn.Close()

	controller, err := conn.Controller()
	if err != nil {
		return err
	}

	controllerConn, err := kafka.Dial("tcp", net.JoinHostPort(controller.Host, strconv.Itoa(controller.Port)))
	if err != nil {
		return err
	}
	defer controllerConn.Close()

	topicConfigs := []kafka.TopicConfig{
		{
			Topic:             GetJobQueueTopic(),
			NumPartitions:     GetPartitions(),
			ReplicationFactor: GetReplicationFactor(),
		},
	}

	return controllerConn.CreateTopics(topicConfigs...)
}

// SendMessage sends a message to the Kafka topic using the producer writer.
func SendMessage(writer *kafka.Writer, key string, value string) error {
	return writer.WriteMessages(context.Background(),
		kafka.Message{
			Key:   []byte(key),
			Value: []byte(value),
		},
	)
}