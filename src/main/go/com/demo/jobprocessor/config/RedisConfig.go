package config

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig configures the Redis connection and provides helper methods
// for common Redis operations with JSON serialization.

var ctx = context.Background()

// GetRedisHost returns the Redis host from env or default.
func GetRedisHost() string {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		return "localhost"
	}
	return host
}

// GetRedisPort returns the Redis port from env or default.
func GetRedisPort() int {
	port := os.Getenv("REDIS_PORT")
	if port == "" {
		return 6379
	}
	val, err := strconv.Atoi(port)
	if err != nil {
		return 6379
	}
	return val
}

// NewRedisClient creates a configured Redis client.
// Equivalent to Java's RedisConnectionFactory + RedisTemplate.
func NewRedisClient() *redis.Client {
	return redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%d", GetRedisHost(), GetRedisPort()),
		DB:   0,
	})
}

// PingRedis checks if the Redis connection is alive.
func PingRedis(client *redis.Client) error {
	return client.Ping(ctx).Err()
}

// SetJSON stores a value as JSON in Redis (mirrors Java's GenericJackson2JsonRedisSerializer).
func SetJSON(client *redis.Client, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return client.Set(ctx, key, data, expiration).Err()
}

// GetJSON retrieves a JSON value from Redis and unmarshals it into the target.
func GetJSON(client *redis.Client, key string, target interface{}) error {
	data, err := client.Get(ctx, key).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// SetHash stores a hash field with JSON value (mirrors Java's HashValueSerializer).
func SetHash(client *redis.Client, key string, field string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return client.HSet(ctx, key, field, data).Err()
}

// GetHash retrieves a hash field and unmarshals the JSON value.
func GetHash(client *redis.Client, key string, field string, target interface{}) error {
	data, err := client.HGet(ctx, key, field).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// Delete removes a key from Redis.
func Delete(client *redis.Client, key string) error {
	return client.Del(ctx, key).Err()
}