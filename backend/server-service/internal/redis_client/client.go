package redis_client

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"

	"server-service/internal/config"
)

// Client wraps Redis client with high-performance pub/sub capabilities
type Client struct {
	rdb    *redis.Client
	config config.RedisConfig
	pubsub *redis.PubSub

	// Event channels for different types of events
	authEvents       chan *Event
	projectEvents    chan *Event
	simulationEvents chan *Event
	simulationData   chan *Event

	// Subscription management
	subscriptions map[string]bool
	mutex         sync.RWMutex

	// Performance monitoring
	messagesReceived  int64
	messagesPublished int64

	// Context for graceful shutdown
	ctx    context.Context
	cancel context.CancelFunc
}

// Event represents a Redis pub/sub event
type Event struct {
	Channel   string    `json:"channel"`
	Pattern   string    `json:"pattern,omitempty"`
	Payload   string    `json:"payload"`
	Timestamp time.Time `json:"timestamp"`
}

// New creates a new Redis client with optimized settings
func New(config config.RedisConfig) (*Client, error) {
	// Create Redis client with performance optimizations
	rdb := redis.NewClient(&redis.Options{
		Addr:         config.Address,
		Password:     config.Password,
		DB:           config.DB,
		PoolSize:     config.PoolSize,
		MaxRetries:   config.MaxRetries,
		DialTimeout:  config.DialTimeout,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		PoolTimeout:  10 * time.Second,
		IdleTimeout:  30 * time.Second,

		// Performance optimizations
		MaxConnAge:         0, // Don't close connections due to age
		IdleCheckFrequency: 60 * time.Second,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := rdb.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	// Create client context
	clientCtx, clientCancel := context.WithCancel(context.Background())

	client := &Client{
		rdb:              rdb,
		config:           config,
		authEvents:       make(chan *Event, 1000), // Buffered channels for high throughput
		projectEvents:    make(chan *Event, 1000),
		simulationEvents: make(chan *Event, 1000),
		simulationData:   make(chan *Event, 10000), // Larger buffer for high-frequency data
		subscriptions:    make(map[string]bool),
		ctx:              clientCtx,
		cancel:           clientCancel,
	}

	log.Printf("Connected to Redis at %s with pool size %d", config.Address, config.PoolSize)
	return client, nil
}

// SubscribeToEvents subscribes to all backend service events
func (c *Client) SubscribeToEvents() error {
	// Subscribe to all event channels
	channels := []string{
		"auth:events:*",       // Auth service events
		"project:events:*",    // Project service events
		"simulation:events:*", // Simulation service events
		"simulation:data:*",   // High-frequency simulation data
	}

	c.pubsub = c.rdb.PSubscribe(c.ctx, channels...)

	// Start message processing goroutine
	go c.processMessages()

	log.Printf("Subscribed to Redis channels: %v", channels)
	return nil
}

// processMessages processes incoming Redis messages
func (c *Client) processMessages() {
	ch := c.pubsub.Channel()

	for {
		select {
		case msg := <-ch:
			if msg == nil {
				continue
			}

			event := &Event{
				Channel:   msg.Channel,
				Pattern:   msg.Pattern,
				Payload:   msg.Payload,
				Timestamp: time.Now(),
			}

			// Route message to appropriate channel based on pattern
			c.routeMessage(event)
			c.messagesReceived++

		case <-c.ctx.Done():
			log.Println("Redis message processing stopped")
			return
		}
	}
}

// routeMessage routes messages to appropriate channels
func (c *Client) routeMessage(event *Event) {
	switch {
	case matchesPattern(event.Channel, "auth:events:*"):
		select {
		case c.authEvents <- event:
		default:
			log.Printf("Auth events channel full, dropping message from %s", event.Channel)
		}

	case matchesPattern(event.Channel, "project:events:*"):
		select {
		case c.projectEvents <- event:
		default:
			log.Printf("Project events channel full, dropping message from %s", event.Channel)
		}

	case matchesPattern(event.Channel, "simulation:events:*"):
		select {
		case c.simulationEvents <- event:
		default:
			log.Printf("Simulation events channel full, dropping message from %s", event.Channel)
		}

	case matchesPattern(event.Channel, "simulation:data:*"):
		select {
		case c.simulationData <- event:
		default:
			log.Printf("Simulation data channel full, dropping message from %s", event.Channel)
		}

	default:
		log.Printf("Unknown channel pattern: %s", event.Channel)
	}
}

// GetAuthEvents returns the auth events channel
func (c *Client) GetAuthEvents() <-chan *Event {
	return c.authEvents
}

// GetProjectEvents returns the project events channel
func (c *Client) GetProjectEvents() <-chan *Event {
	return c.projectEvents
}

// GetSimulationEvents returns the simulation events channel
func (c *Client) GetSimulationEvents() <-chan *Event {
	return c.simulationEvents
}

// GetSimulationData returns the simulation data channel
func (c *Client) GetSimulationData() <-chan *Event {
	return c.simulationData
}

// Publish publishes a message to a Redis channel
func (c *Client) Publish(channel, message string) error {
	ctx, cancel := context.WithTimeout(c.ctx, 1*time.Second)
	defer cancel()

	if err := c.rdb.Publish(ctx, channel, message).Err(); err != nil {
		return fmt.Errorf("failed to publish to channel %s: %w", channel, err)
	}

	c.messagesPublished++
	return nil
}

// PublishJSON publishes a JSON message to a Redis channel
func (c *Client) PublishJSON(channel string, data interface{}) error {
	// Use standard JSON encoding
	jsonData, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return c.Publish(channel, string(jsonData))
}

// GetStats returns Redis client statistics
func (c *Client) GetStats() map[string]interface{} {
	stats := c.rdb.PoolStats()

	return map[string]interface{}{
		"messages_received":  c.messagesReceived,
		"messages_published": c.messagesPublished,
		"pool_stats": map[string]interface{}{
			"hits":        stats.Hits,
			"misses":      stats.Misses,
			"timeouts":    stats.Timeouts,
			"total_conns": stats.TotalConns,
			"idle_conns":  stats.IdleConns,
			"stale_conns": stats.StaleConns,
		},
		"subscriptions": len(c.subscriptions),
	}
}

// HealthCheck checks Redis connection health
func (c *Client) HealthCheck() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	return c.rdb.Ping(ctx).Err() == nil
}

// Close closes the Redis client and all subscriptions
func (c *Client) Close() error {
	log.Println("Closing Redis client...")

	// Cancel context to stop message processing
	c.cancel()

	// Close pub/sub
	if c.pubsub != nil {
		if err := c.pubsub.Close(); err != nil {
			log.Printf("Error closing pubsub: %v", err)
		}
	}

	// Close Redis client
	if err := c.rdb.Close(); err != nil {
		return fmt.Errorf("failed to close Redis client: %w", err)
	}

	// Close channels
	close(c.authEvents)
	close(c.projectEvents)
	close(c.simulationEvents)
	close(c.simulationData)

	log.Println("Redis client closed")
	return nil
}

// matchesPattern checks if a channel matches a pattern
func matchesPattern(channel, pattern string) bool {
	// Simple pattern matching for Redis patterns
	// This is a simplified version - Redis uses more complex pattern matching
	if pattern == "*" {
		return true
	}

	// Remove the * from the end and check prefix
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		return len(channel) >= len(prefix) && channel[:len(prefix)] == prefix
	}

	return channel == pattern
}

// GetChannelStats returns statistics for each channel type
func (c *Client) GetChannelStats() map[string]interface{} {
	return map[string]interface{}{
		"auth_events": map[string]interface{}{
			"buffer_size":    cap(c.authEvents),
			"current_length": len(c.authEvents),
			"utilization":    float64(len(c.authEvents)) / float64(cap(c.authEvents)),
		},
		"project_events": map[string]interface{}{
			"buffer_size":    cap(c.projectEvents),
			"current_length": len(c.projectEvents),
			"utilization":    float64(len(c.projectEvents)) / float64(cap(c.projectEvents)),
		},
		"simulation_events": map[string]interface{}{
			"buffer_size":    cap(c.simulationEvents),
			"current_length": len(c.simulationEvents),
			"utilization":    float64(len(c.simulationEvents)) / float64(cap(c.simulationEvents)),
		},
		"simulation_data": map[string]interface{}{
			"buffer_size":    cap(c.simulationData),
			"current_length": len(c.simulationData),
			"utilization":    float64(len(c.simulationData)) / float64(cap(c.simulationData)),
		},
	}
}
