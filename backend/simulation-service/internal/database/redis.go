package database

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/systemsim/simulation-service/internal/config"
)

// RedisClient wraps the Redis client with additional functionality
type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisClient creates a new Redis client connection
func NewRedisClient(cfg config.RedisConfig) (*RedisClient, error) {
	rdb := redis.NewClient(&redis.Options{
		Addr:     cfg.GetRedisAddr(),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: 20,
		MinIdleConns: 5,
		MaxRetries: 3,
		RetryDelay: time.Millisecond * 100,
		DialTimeout: time.Second * 5,
		ReadTimeout: time.Second * 3,
		WriteTimeout: time.Second * 3,
	})

	ctx := context.Background()

	// Test connection
	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisClient{
		client: rdb,
		ctx:    ctx,
	}, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	return r.client.Close()
}

// GetClient returns the underlying Redis client
func (r *RedisClient) GetClient() *redis.Client {
	return r.client
}

// GetContext returns the context used for Redis operations
func (r *RedisClient) GetContext() context.Context {
	return r.ctx
}

// Ping tests the Redis connection
func (r *RedisClient) Ping() error {
	return r.client.Ping(r.ctx).Err()
}

// Set sets a key-value pair with optional expiration
func (r *RedisClient) Set(key string, value interface{}, expiration time.Duration) error {
	return r.client.Set(r.ctx, key, value, expiration).Err()
}

// Get gets a value by key
func (r *RedisClient) Get(key string) (string, error) {
	return r.client.Get(r.ctx, key).Result()
}

// Del deletes one or more keys
func (r *RedisClient) Del(keys ...string) error {
	return r.client.Del(r.ctx, keys...).Err()
}

// Exists checks if a key exists
func (r *RedisClient) Exists(key string) (bool, error) {
	result, err := r.client.Exists(r.ctx, key).Result()
	return result > 0, err
}

// HSet sets a hash field
func (r *RedisClient) HSet(key, field string, value interface{}) error {
	return r.client.HSet(r.ctx, key, field, value).Err()
}

// HGet gets a hash field value
func (r *RedisClient) HGet(key, field string) (string, error) {
	return r.client.HGet(r.ctx, key, field).Result()
}

// HGetAll gets all hash fields and values
func (r *RedisClient) HGetAll(key string) (map[string]string, error) {
	return r.client.HGetAll(r.ctx, key).Result()
}

// HDel deletes hash fields
func (r *RedisClient) HDel(key string, fields ...string) error {
	return r.client.HDel(r.ctx, key, fields...).Err()
}

// LPush pushes elements to the left of a list
func (r *RedisClient) LPush(key string, values ...interface{}) error {
	return r.client.LPush(r.ctx, key, values...).Err()
}

// RPop pops an element from the right of a list
func (r *RedisClient) RPop(key string) (string, error) {
	return r.client.RPop(r.ctx, key).Result()
}

// LLen gets the length of a list
func (r *RedisClient) LLen(key string) (int64, error) {
	return r.client.LLen(r.ctx, key).Result()
}

// Publish publishes a message to a channel
func (r *RedisClient) Publish(channel string, message interface{}) error {
	return r.client.Publish(r.ctx, channel, message).Err()
}

// Subscribe subscribes to channels
func (r *RedisClient) Subscribe(channels ...string) *redis.PubSub {
	return r.client.Subscribe(r.ctx, channels...)
}

// ZAdd adds members to a sorted set
func (r *RedisClient) ZAdd(key string, members ...redis.Z) error {
	return r.client.ZAdd(r.ctx, key, members...).Err()
}

// ZRange gets members from a sorted set by range
func (r *RedisClient) ZRange(key string, start, stop int64) ([]string, error) {
	return r.client.ZRange(r.ctx, key, start, stop).Result()
}

// ZRangeWithScores gets members with scores from a sorted set by range
func (r *RedisClient) ZRangeWithScores(key string, start, stop int64) ([]redis.Z, error) {
	return r.client.ZRangeWithScores(r.ctx, key, start, stop).Result()
}

// ZRem removes members from a sorted set
func (r *RedisClient) ZRem(key string, members ...interface{}) error {
	return r.client.ZRem(r.ctx, key, members...).Err()
}

// Incr increments a key's value
func (r *RedisClient) Incr(key string) (int64, error) {
	return r.client.Incr(r.ctx, key).Result()
}

// IncrBy increments a key's value by a specific amount
func (r *RedisClient) IncrBy(key string, value int64) (int64, error) {
	return r.client.IncrBy(r.ctx, key, value).Result()
}

// Expire sets a key's expiration
func (r *RedisClient) Expire(key string, expiration time.Duration) error {
	return r.client.Expire(r.ctx, key, expiration).Err()
}

// TTL gets a key's time to live
func (r *RedisClient) TTL(key string) (time.Duration, error) {
	return r.client.TTL(r.ctx, key).Result()
}

// Keys gets all keys matching a pattern
func (r *RedisClient) Keys(pattern string) ([]string, error) {
	return r.client.Keys(r.ctx, pattern).Result()
}

// FlushDB flushes the current database
func (r *RedisClient) FlushDB() error {
	return r.client.FlushDB(r.ctx).Err()
}

// Pipeline creates a new pipeline for batch operations
func (r *RedisClient) Pipeline() redis.Pipeliner {
	return r.client.Pipeline()
}

// TxPipeline creates a new transaction pipeline
func (r *RedisClient) TxPipeline() redis.Pipeliner {
	return r.client.TxPipeline()
}
