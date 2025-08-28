package health

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisHealthMessage represents a health message from Redis
type RedisHealthMessage struct {
	Service   string    `json:"service"`
	Status    string    `json:"status"`
	Timestamp time.Time `json:"timestamp"`
	Details   string    `json:"details,omitempty"`
}

// RedisHealthSubscriber subscribes to health updates from Redis
type RedisHealthSubscriber struct {
	client    *redis.Client
	callbacks []func(string, ServiceHealth)
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewRedisHealthSubscriber creates a new Redis health subscriber
func NewRedisHealthSubscriber(redisAddr, redisPassword string, redisDB int) (*RedisHealthSubscriber, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())

	subscriber := &RedisHealthSubscriber{
		client:    client,
		callbacks: make([]func(string, ServiceHealth), 0),
		ctx:       ctx,
		cancel:    cancel,
	}

	// Start subscription goroutine
	go subscriber.subscribeToHealthUpdates()

	log.Printf("Redis health subscriber connected to %s", redisAddr)
	return subscriber, nil
}

// OnHealthChange registers a callback for health status changes
func (rhs *RedisHealthSubscriber) OnHealthChange(callback func(string, ServiceHealth)) {
	rhs.callbacks = append(rhs.callbacks, callback)
}

// subscribeToHealthUpdates subscribes to Redis health channels
func (rhs *RedisHealthSubscriber) subscribeToHealthUpdates() {
	pubsub := rhs.client.PSubscribe(rhs.ctx, "service:health:*")
	defer pubsub.Close()

	log.Printf("Subscribed to Redis health updates: service:health:*")

	for {
		select {
		case <-rhs.ctx.Done():
			return
		default:
			msg, err := pubsub.ReceiveMessage(rhs.ctx)
			if err != nil {
				if rhs.ctx.Err() != nil {
					return // Context cancelled
				}
				log.Printf("Error receiving Redis message: %v", err)
				time.Sleep(time.Second)
				continue
			}

			rhs.handleHealthMessage(msg)
		}
	}
}

// handleHealthMessage processes a health message from Redis
func (rhs *RedisHealthSubscriber) handleHealthMessage(msg *redis.Message) {
	var healthMsg RedisHealthMessage
	if err := json.Unmarshal([]byte(msg.Payload), &healthMsg); err != nil {
		log.Printf("Failed to unmarshal health message: %v", err)
		return
	}

	// Convert to ServiceHealth
	serviceHealth := ServiceHealth{
		Service:     healthMsg.Service,
		Status:      healthMsg.Status,
		LastChecked: healthMsg.Timestamp,
		Details:     healthMsg.Details,
	}

	log.Printf("Received health update from Redis: %s = %s", healthMsg.Service, healthMsg.Status)

	// Notify callbacks
	for _, callback := range rhs.callbacks {
		go callback(healthMsg.Service, serviceHealth)
	}
}

// Close stops the Redis subscriber
func (rhs *RedisHealthSubscriber) Close() {
	rhs.cancel()
	rhs.client.Close()
	log.Printf("Redis health subscriber closed")
}

// RedisHealthPublisher publishes health updates to Redis
type RedisHealthPublisher struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisHealthPublisher creates a new Redis health publisher
func NewRedisHealthPublisher(redisAddr, redisPassword string, redisDB int) (*RedisHealthPublisher, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	// Test connection
	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	publisher := &RedisHealthPublisher{
		client: client,
		ctx:    context.Background(),
	}

	log.Printf("Redis health publisher connected to %s", redisAddr)
	return publisher, nil
}

// PublishHealth publishes a health status update
func (rhp *RedisHealthPublisher) PublishHealth(service, status, details string) error {
	healthMsg := RedisHealthMessage{
		Service:   service,
		Status:    status,
		Timestamp: time.Now(),
		Details:   details,
	}

	data, err := json.Marshal(healthMsg)
	if err != nil {
		return err
	}

	channel := "service:health:" + service
	if err := rhp.client.Publish(rhp.ctx, channel, data).Err(); err != nil {
		return err
	}

	log.Printf("Published health update: %s = %s", service, status)
	return nil
}

// Close closes the Redis publisher
func (rhp *RedisHealthPublisher) Close() {
	rhp.client.Close()
	log.Printf("Redis health publisher closed")
}
