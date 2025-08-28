package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Subscriber handles subscribing to Redis events
type Subscriber struct {
	redis  *redis.Client
	ctx    context.Context
	cancel context.CancelFunc
	pubsub *redis.PubSub
}

// NewSubscriber creates a new event subscriber
func NewSubscriber(redisClient *redis.Client) *Subscriber {
	ctx, cancel := context.WithCancel(context.Background())
	return &Subscriber{
		redis:  redisClient,
		ctx:    ctx,
		cancel: cancel,
	}
}

// EmailProcessor defines the interface for processing email tasks
type EmailProcessor interface {
	ProcessWelcomeEmail(task *EmailTask) error
	ProcessVerificationEmail(task *EmailTask) error
	ProcessPasswordResetEmail(task *EmailTask) error
	ProcessNotificationEmail(task *EmailTask) error
}

// Start starts the subscriber and begins processing events
func (s *Subscriber) Start(emailProcessor EmailProcessor) error {
	// Subscribe to channels
	s.pubsub = s.redis.Subscribe(s.ctx,
		ChannelAsyncEmailQueue,        // Email processing
		ChannelSystemAnnouncements,    // System announcements
	)

	// Wait for subscription confirmation
	_, err := s.pubsub.Receive(s.ctx)
	if err != nil {
		return fmt.Errorf("failed to subscribe to channels: %w", err)
	}

	log.Println("Event subscriber started, listening for events...")

	// Start processing messages
	go s.processMessages(emailProcessor)

	return nil
}

// Stop stops the subscriber
func (s *Subscriber) Stop() error {
	log.Println("Stopping event subscriber...")
	
	if s.pubsub != nil {
		s.pubsub.Close()
	}
	
	s.cancel()
	return nil
}

// processMessages processes incoming messages from Redis
func (s *Subscriber) processMessages(emailProcessor EmailProcessor) {
	ch := s.pubsub.Channel()
	
	for {
		select {
		case msg := <-ch:
			if msg == nil {
				continue
			}
			
			if err := s.handleMessage(msg, emailProcessor); err != nil {
				log.Printf("Error processing message from channel %s: %v", msg.Channel, err)
			}
			
		case <-s.ctx.Done():
			log.Println("Event subscriber context cancelled, stopping message processing")
			return
		}
	}
}

// handleMessage handles a single message based on its channel
func (s *Subscriber) handleMessage(msg *redis.Message, emailProcessor EmailProcessor) error {
	switch msg.Channel {
	case ChannelAsyncEmailQueue:
		return s.handleEmailTask(msg.Payload, emailProcessor)
	case ChannelSystemAnnouncements:
		return s.handleSystemAnnouncement(msg.Payload)
	default:
		log.Printf("Unknown channel: %s", msg.Channel)
		return nil
	}
}

// handleEmailTask processes email tasks
func (s *Subscriber) handleEmailTask(payload string, emailProcessor EmailProcessor) error {
	var task EmailTask
	if err := json.Unmarshal([]byte(payload), &task); err != nil {
		return fmt.Errorf("failed to unmarshal email task: %w", err)
	}

	log.Printf("Processing email task: type=%s, to=%s, priority=%d", task.Type, task.To, task.Priority)

	var err error
	switch task.Type {
	case "welcome":
		err = emailProcessor.ProcessWelcomeEmail(&task)
	case "verification":
		err = emailProcessor.ProcessVerificationEmail(&task)
	case "password_reset":
		err = emailProcessor.ProcessPasswordResetEmail(&task)
	case "notification":
		err = emailProcessor.ProcessNotificationEmail(&task)
	default:
		return fmt.Errorf("unknown email task type: %s", task.Type)
	}

	if err != nil {
		// Handle retry logic
		if task.Retries < task.MaxRetries {
			task.Retries++
			log.Printf("Email task failed, retrying (%d/%d): %v", task.Retries, task.MaxRetries, err)
			
			// Republish with delay
			go func() {
				time.Sleep(time.Duration(task.Retries) * time.Minute) // Exponential backoff
				if retryErr := s.republishEmailTask(&task); retryErr != nil {
					log.Printf("Failed to republish email task: %v", retryErr)
				}
			}()
		} else {
			log.Printf("Email task failed permanently after %d retries: %v", task.MaxRetries, err)
		}
		return err
	}

	log.Printf("Email task completed successfully: type=%s, to=%s", task.Type, task.To)
	return nil
}

// handleSystemAnnouncement processes system announcements
func (s *Subscriber) handleSystemAnnouncement(payload string) error {
	var announcement SystemAnnouncement
	if err := json.Unmarshal([]byte(payload), &announcement); err != nil {
		return fmt.Errorf("failed to unmarshal system announcement: %w", err)
	}

	log.Printf("Received system announcement: type=%s, severity=%s, message=%s", 
		announcement.Type, announcement.Severity, announcement.Message)

	// Handle different types of announcements
	switch announcement.Type {
	case "maintenance":
		log.Printf("MAINTENANCE ALERT: %s", announcement.Message)
	case "security_alert":
		log.Printf("SECURITY ALERT: %s", announcement.Message)
	case "feature_update":
		log.Printf("FEATURE UPDATE: %s", announcement.Message)
	default:
		log.Printf("SYSTEM MESSAGE: %s", announcement.Message)
	}

	return nil
}

// republishEmailTask republishes a failed email task for retry
func (s *Subscriber) republishEmailTask(task *EmailTask) error {
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal email task for retry: %w", err)
	}

	return s.redis.Publish(s.ctx, ChannelAsyncEmailQueue, data).Err()
}

// Health check for subscriber
func (s *Subscriber) HealthCheck() error {
	if s.pubsub == nil {
		return fmt.Errorf("subscriber not started")
	}

	ctx, cancel := context.WithTimeout(s.ctx, 5*time.Second)
	defer cancel()
	
	return s.redis.Ping(ctx).Err()
}

// GetSubscribedChannels returns the list of subscribed channels
func (s *Subscriber) GetSubscribedChannels() []string {
	return []string{
		ChannelAsyncEmailQueue,
		ChannelSystemAnnouncements,
	}
}

// MockEmailProcessor is a simple implementation for testing
type MockEmailProcessor struct{}

func (m *MockEmailProcessor) ProcessWelcomeEmail(task *EmailTask) error {
	log.Printf("MOCK: Sending welcome email to %s", task.To)
	// Simulate email sending delay
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (m *MockEmailProcessor) ProcessVerificationEmail(task *EmailTask) error {
	log.Printf("MOCK: Sending verification email to %s with token %s", 
		task.To, task.Variables["verification_token"])
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (m *MockEmailProcessor) ProcessPasswordResetEmail(task *EmailTask) error {
	log.Printf("MOCK: Sending password reset email to %s with token %s", 
		task.To, task.Variables["reset_token"])
	time.Sleep(100 * time.Millisecond)
	return nil
}

func (m *MockEmailProcessor) ProcessNotificationEmail(task *EmailTask) error {
	log.Printf("MOCK: Sending notification email to %s: %s", task.To, task.Subject)
	time.Sleep(100 * time.Millisecond)
	return nil
}
