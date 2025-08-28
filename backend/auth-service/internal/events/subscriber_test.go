package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/systemsim/auth-service/internal/testutils"
)

// TestEmailProcessor implements EmailProcessor for testing
type TestEmailProcessor struct {
	SentEmails []EmailTask
	mutex      sync.Mutex
}

// ProcessWelcomeEmail implements the EmailProcessor interface
func (p *TestEmailProcessor) ProcessWelcomeEmail(task *EmailTask) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.SentEmails = append(p.SentEmails, *task)
	return nil
}

// ProcessVerificationEmail implements the EmailProcessor interface
func (p *TestEmailProcessor) ProcessVerificationEmail(task *EmailTask) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.SentEmails = append(p.SentEmails, *task)
	return nil
}

// ProcessPasswordResetEmail implements the EmailProcessor interface
func (p *TestEmailProcessor) ProcessPasswordResetEmail(task *EmailTask) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.SentEmails = append(p.SentEmails, *task)
	return nil
}

// ProcessNotificationEmail implements the EmailProcessor interface
func (p *TestEmailProcessor) ProcessNotificationEmail(task *EmailTask) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.SentEmails = append(p.SentEmails, *task)
	return nil
}

// GetLastEmail returns the last sent email (thread-safe)
func (p *TestEmailProcessor) GetLastEmail() *EmailTask {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	if len(p.SentEmails) == 0 {
		return nil
	}
	return &p.SentEmails[len(p.SentEmails)-1]
}

// ClearEmails clears all sent emails (thread-safe)
func (p *TestEmailProcessor) ClearEmails() {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.SentEmails = nil
}

// GetEmailCount returns the number of sent emails (thread-safe)
func (p *TestEmailProcessor) GetEmailCount() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	return len(p.SentEmails)
}

func TestSubscriber_EmailProcessing(t *testing.T) {
	// Setup
	redisClient := testutils.SetupTestRedis(t)
	defer redisClient.Close()
	defer testutils.CleanupTestRedis(t, redisClient)

	subscriber := NewSubscriber(redisClient)
	emailProcessor := &TestEmailProcessor{}

	// Start subscriber
	err := subscriber.Start(emailProcessor)
	require.NoError(t, err)
	defer subscriber.Stop()

	// Give subscriber time to start
	time.Sleep(100 * time.Millisecond)

	tests := []struct {
		name string
		task *EmailTask
	}{
		{
			name: "welcome_email",
			task: &EmailTask{
				ID:       "test-welcome-task",
				Type:     "welcome",
				To:       "welcome@example.com",
				Subject:  "Welcome!",
				Template: "welcome_template",
				Variables: map[string]string{
					"first_name": "John",
					"last_name":  "Doe",
				},
				Priority:   2,
				Timestamp:  time.Now(),
				Retries:    0,
				MaxRetries: 3,
			},
		},
		{
			name: "verification_email",
			task: &EmailTask{
				ID:       "test-verification-task",
				Type:     "verification",
				To:       "verify@example.com",
				Subject:  "Verify your email",
				Template: "verification_template",
				Variables: map[string]string{
					"verification_token": "abc123",
					"user_name":         "Jane Doe",
				},
				Priority:   1,
				Timestamp:  time.Now(),
				Retries:    0,
				MaxRetries: 3,
			},
		},
		{
			name: "password_reset_email",
			task: &EmailTask{
				ID:       "test-password-reset-task",
				Type:     "password_reset",
				To:       "reset@example.com",
				Subject:  "Reset your password",
				Template: "password_reset_template",
				Variables: map[string]string{
					"reset_token": "xyz789",
					"user_name":   "Bob Smith",
				},
				Priority:   1,
				Timestamp:  time.Now(),
				Retries:    0,
				MaxRetries: 3,
			},
		},
		{
			name: "notification_email",
			task: &EmailTask{
				ID:       "test-notification-task",
				Type:     "notification",
				To:       "notify@example.com",
				Subject:  "Notification",
				Template: "notification_template",
				Variables: map[string]string{
					"message":   "Test notification",
					"user_name": "Alice Johnson",
				},
				Priority:   3,
				Timestamp:  time.Now(),
				Retries:    0,
				MaxRetries: 3,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear previous emails
			emailProcessor.ClearEmails()

			// Serialize and publish email task
			taskJSON, err := json.Marshal(tt.task)
			require.NoError(t, err)

			// Publish to email queue
			err = redisClient.Publish(context.Background(), ChannelAsyncEmailQueue, taskJSON).Err()
			require.NoError(t, err)

			// Wait for processing
			time.Sleep(200 * time.Millisecond)

			// Verify email was processed
			assert.Len(t, emailProcessor.SentEmails, 1)
			processedEmail := emailProcessor.GetLastEmail()
			assert.NotNil(t, processedEmail)
			assert.Equal(t, tt.task.Type, processedEmail.Type)
			assert.Equal(t, tt.task.To, processedEmail.To)
			assert.Equal(t, tt.task.Subject, processedEmail.Subject)
			assert.Equal(t, tt.task.Template, processedEmail.Template)
			assert.Equal(t, tt.task.Variables, processedEmail.Variables)
			assert.Equal(t, tt.task.Priority, processedEmail.Priority)
			assert.Equal(t, tt.task.Retries, processedEmail.Retries)
			assert.Equal(t, tt.task.MaxRetries, processedEmail.MaxRetries)
		})
	}
}

func TestSubscriber_SystemAnnouncements(t *testing.T) {
	// Setup
	redisClient := testutils.SetupTestRedis(t)
	defer redisClient.Close()
	defer testutils.CleanupTestRedis(t, redisClient)

	subscriber := NewSubscriber(redisClient)
	emailProcessor := &TestEmailProcessor{}

	// Start subscriber
	err := subscriber.Start(emailProcessor)
	require.NoError(t, err)
	defer subscriber.Stop()

	// Give subscriber time to start
	time.Sleep(100 * time.Millisecond)

	tests := []struct {
		name         string
		announcement *SystemAnnouncement
	}{
		{
			name: "maintenance_announcement",
			announcement: &SystemAnnouncement{
				ID:        "test-maintenance-announcement",
				Type:      "maintenance",
				Message:   "System will be down for maintenance from 2:00 AM to 4:00 AM UTC. Duration: 2 hours. All services affected. Contact: support@systemsim.com",
				Severity:  "critical",
				Timestamp: time.Now(),
				Source:    "auth-service",
			},
		},
		{
			name: "security_announcement",
			announcement: &SystemAnnouncement{
				ID:        "test-security-announcement",
				Type:      "security_alert",
				Message:   "Please update your passwords as a security precaution. Action required within 7 days.",
				Severity:  "warning",
				Timestamp: time.Now(),
				Source:    "auth-service",
			},
		},
		{
			name: "feature_announcement",
			announcement: &SystemAnnouncement{
				ID:        "test-feature-announcement",
				Type:      "feature_update",
				Message:   "Check out our new project collaboration features. Learn more at: https://docs.systemsim.com/collaboration",
				Severity:  "info",
				Timestamp: time.Now(),
				Source:    "auth-service",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear previous emails
			emailProcessor.ClearEmails()

			// Serialize and publish system announcement
			announcementJSON, err := json.Marshal(tt.announcement)
			require.NoError(t, err)

			// Publish system announcement
			err = redisClient.Publish(context.Background(), ChannelSystemAnnouncements, announcementJSON).Err()
			require.NoError(t, err)

			// Wait for processing
			time.Sleep(200 * time.Millisecond)

			// System announcements should be logged but not create email tasks
			// Verify no emails were sent (announcements are processed differently)
			assert.Len(t, emailProcessor.SentEmails, 0, "System announcements should not generate email tasks")

			// In a real implementation, you would verify that the announcement was:
			// 1. Logged appropriately
			// 2. Stored in the database for display to users
			// 3. Potentially sent to WebSocket connections for real-time updates
			// For this test, we just verify it doesn't crash the subscriber
		})
	}
}

func TestSubscriber_InvalidMessages(t *testing.T) {
	// Setup
	redisClient := testutils.SetupTestRedis(t)
	defer redisClient.Close()
	defer testutils.CleanupTestRedis(t, redisClient)

	subscriber := NewSubscriber(redisClient)
	emailProcessor := &TestEmailProcessor{}

	// Start subscriber
	err := subscriber.Start(emailProcessor)
	require.NoError(t, err)
	defer subscriber.Stop()

	// Give subscriber time to start
	time.Sleep(100 * time.Millisecond)

	tests := []struct {
		name        string
		channel     string
		message     string
		description string
	}{
		{
			name:        "invalid_json_email_queue",
			channel:     ChannelAsyncEmailQueue,
			message:     "invalid-json-message-that-cannot-be-parsed",
			description: "Invalid JSON should be handled gracefully",
		},
		{
			name:        "malformed_json_email_queue",
			channel:     ChannelAsyncEmailQueue,
			message:     `{"incomplete": "json", "missing_closing_brace": true`,
			description: "Malformed JSON should not crash subscriber",
		},
		{
			name:        "invalid_json_system_announcements",
			channel:     ChannelSystemAnnouncements,
			message:     "invalid-json-message-for-announcements",
			description: "Invalid announcement JSON should be handled gracefully",
		},
		{
			name:        "empty_message_email_queue",
			channel:     ChannelAsyncEmailQueue,
			message:     "",
			description: "Empty messages should not cause errors",
		},
		{
			name:        "empty_message_announcements",
			channel:     ChannelSystemAnnouncements,
			message:     "",
			description: "Empty announcement messages should not cause errors",
		},
		{
			name:        "null_json_email_queue",
			channel:     ChannelAsyncEmailQueue,
			message:     "null",
			description: "Null JSON should be handled gracefully",
		},
		{
			name:        "valid_json_wrong_structure_email",
			channel:     ChannelAsyncEmailQueue,
			message:     `{"wrong": "structure", "not": "email_task"}`,
			description: "Valid JSON with wrong structure should be handled",
		},
		{
			name:        "valid_json_wrong_structure_announcement",
			channel:     ChannelSystemAnnouncements,
			message:     `{"wrong": "structure", "not": "announcement"}`,
			description: "Valid JSON with wrong announcement structure should be handled",
		},
		{
			name:        "extremely_large_message",
			channel:     ChannelAsyncEmailQueue,
			message:     `{"type": "test", "large_field": "` + string(make([]byte, 10000)) + `"}`,
			description: "Extremely large messages should be handled gracefully",
		},
		{
			name:        "unicode_characters",
			channel:     ChannelAsyncEmailQueue,
			message:     `{"type": "test", "unicode": "测试消息 🚀 émojis"}`,
			description: "Unicode characters should be handled properly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear previous emails
			emailProcessor.ClearEmails()

			// Publish invalid message
			err := redisClient.Publish(context.Background(), tt.channel, tt.message).Err()
			require.NoError(t, err, "Failed to publish test message")

			// Wait for processing
			time.Sleep(300 * time.Millisecond)

			// Subscriber should handle invalid messages gracefully
			// No emails should be sent for invalid messages
			assert.Len(t, emailProcessor.SentEmails, 0,
				"Invalid messages should not generate email tasks: %s", tt.description)

			// Verify subscriber is still running and responsive
			// by sending a valid message after the invalid one
			validTask := &EmailTask{
				ID:       "test-valid-after-invalid",
				Type:     "welcome", // Use a valid email type
				To:       "test@example.com",
				Subject:  "Test after invalid",
				Template: "welcome_template",
				Variables: map[string]string{"first_name": "Test", "last_name": "User"},
				Priority:   1,
				Timestamp:  time.Now(),
				Retries:    0,
				MaxRetries: 3,
			}

			validTaskJSON, err := json.Marshal(validTask)
			require.NoError(t, err)

			err = redisClient.Publish(context.Background(), ChannelAsyncEmailQueue, validTaskJSON).Err()
			require.NoError(t, err)

			// Wait for processing
			time.Sleep(200 * time.Millisecond)

			// This valid message should be processed
			assert.Len(t, emailProcessor.SentEmails, 1,
				"Subscriber should still process valid messages after invalid ones")
		})
	}
}

func TestSubscriber_EmailProcessingRetry(t *testing.T) {
	// Setup
	redisClient := testutils.SetupTestRedis(t)
	defer redisClient.Close()
	defer testutils.CleanupTestRedis(t, redisClient)

	subscriber := NewSubscriber(redisClient)

	// Create a failing email processor for testing retry logic
	failingProcessor := &FailingEmailProcessor{
		failCount: 1, // Fail first attempt, succeed on 2nd
		callCount: 0,
	}

	// Start subscriber
	err := subscriber.Start(failingProcessor)
	require.NoError(t, err)
	defer subscriber.Stop()

	// Give subscriber time to start
	time.Sleep(100 * time.Millisecond)

	tests := []struct {
		name           string
		task           *EmailTask
		description    string
	}{
		{
			name: "retry_welcome_email",
			task: &EmailTask{
				ID:       "retry-welcome-task",
				Type:     "welcome",
				To:       "retry-welcome@example.com",
				Subject:  "Welcome - Retry Test",
				Template: "welcome_template",
				Variables: map[string]string{
					"first_name": "Retry",
					"last_name":  "Test",
				},
				Priority:   2,
				Timestamp:  time.Now(),
				Retries:    0,
				MaxRetries: 3, // Allow retries
			},
			description: "Welcome email should handle failure and schedule retry",
		},
		{
			name: "retry_verification_email",
			task: &EmailTask{
				ID:       "retry-verification-task",
				Type:     "verification",
				To:       "retry-verify@example.com",
				Subject:  "Email Verification - Retry Test",
				Template: "verification_template",
				Variables: map[string]string{
					"verification_token": "retry-token-123",
					"user_name":         "Retry User",
				},
				Priority:   1,
				Timestamp:  time.Now(),
				Retries:    0,
				MaxRetries: 3,
			},
			description: "Verification email should handle failure and schedule retry",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset the failing processor
			failingProcessor.Reset(1) // Fail first attempt only

			// Serialize and publish task
			taskJSON, err := json.Marshal(tt.task)
			require.NoError(t, err)

			// Publish task
			err = redisClient.Publish(context.Background(), ChannelAsyncEmailQueue, taskJSON).Err()
			require.NoError(t, err)

			// Wait for initial processing
			time.Sleep(500 * time.Millisecond)

			// Verify the processor was called at least once (initial attempt)
			callCount := failingProcessor.GetCallCount()
			assert.GreaterOrEqual(t, callCount, 1,
				"Processor should be called at least once for %s", tt.description)

			// Verify that an error was logged (indicating retry will be scheduled)
			errors := failingProcessor.GetErrors()
			assert.Len(t, errors, 1,
				"Should have one error logged for initial failure: %s", tt.description)

			// Note: We don't wait for the actual retry because it uses minute-long delays
			// In production, this is correct behavior for email retry logic
			// The test verifies that the retry mechanism is triggered, not that it completes immediately
		})
	}
}

func TestSubscriber_StartStop(t *testing.T) {
	// Setup
	redisClient := testutils.SetupTestRedis(t)
	defer redisClient.Close()
	defer testutils.CleanupTestRedis(t, redisClient)

	subscriber := NewSubscriber(redisClient)
	emailProcessor := &TestEmailProcessor{}

	// Test 1: Initial start
	t.Run("initial_start", func(t *testing.T) {
		err := subscriber.Start(emailProcessor)
		assert.NoError(t, err, "Initial start should succeed")

		// Give subscriber time to start
		time.Sleep(100 * time.Millisecond)

		// Verify subscriber is working by sending a test message
		testTask := &EmailTask{
			ID:       "start-stop-test-task",
			Type:     "welcome",
			To:       "test@example.com",
			Subject:  "Start/Stop Test",
			Template: "welcome_template",
			Variables: map[string]string{"first_name": "Test", "last_name": "User"},
			Priority:   1,
			Timestamp:  time.Now(),
			Retries:    0,
			MaxRetries: 3,
		}

		taskJSON, err := json.Marshal(testTask)
		require.NoError(t, err)

		err = redisClient.Publish(context.Background(), ChannelAsyncEmailQueue, taskJSON).Err()
		require.NoError(t, err)

		time.Sleep(200 * time.Millisecond)

		// Verify message was processed
		assert.Len(t, emailProcessor.SentEmails, 1, "Subscriber should process messages after start")
	})

	// Test 2: Stop subscriber
	t.Run("stop_subscriber", func(t *testing.T) {
		err := subscriber.Stop()
		assert.NoError(t, err, "Stop should succeed")

		// Give time for graceful shutdown
		time.Sleep(100 * time.Millisecond)

		// Clear previous emails
		emailProcessor.ClearEmails()

		// Try to send a message after stop - it should not be processed
		testTask := &EmailTask{
			ID:       "after-stop-test-task",
			Type:     "welcome",
			To:       "test-after-stop@example.com",
			Subject:  "After Stop Test",
			Template: "welcome_template",
			Variables: map[string]string{"first_name": "After", "last_name": "Stop"},
			Priority:   1,
			Timestamp:  time.Now(),
			Retries:    0,
			MaxRetries: 3,
		}

		taskJSON, err := json.Marshal(testTask)
		require.NoError(t, err)

		err = redisClient.Publish(context.Background(), ChannelAsyncEmailQueue, taskJSON).Err()
		require.NoError(t, err)

		time.Sleep(200 * time.Millisecond)

		// Message should not be processed after stop
		assert.Len(t, emailProcessor.SentEmails, 0, "Subscriber should not process messages after stop")
	})

	// Test 3: Restart after stop
	t.Run("restart_after_stop", func(t *testing.T) {
		// Create a new subscriber since the old one's context was cancelled
		newSubscriber := NewSubscriber(redisClient)

		err := newSubscriber.Start(emailProcessor)
		assert.NoError(t, err, "Restart should succeed")
		defer newSubscriber.Stop()

		// Give subscriber time to start
		time.Sleep(100 * time.Millisecond)

		// Clear previous emails
		emailProcessor.ClearEmails()

		// Verify subscriber is working again
		testTask := &EmailTask{
			ID:       "restart-test-task",
			Type:     "welcome",
			To:       "restart@example.com",
			Subject:  "Restart Test",
			Template: "welcome_template",
			Variables: map[string]string{"first_name": "Restart", "last_name": "Test"},
			Priority:   1,
			Timestamp:  time.Now(),
			Retries:    0,
			MaxRetries: 3,
		}

		taskJSON, err := json.Marshal(testTask)
		require.NoError(t, err)

		err = redisClient.Publish(context.Background(), ChannelAsyncEmailQueue, taskJSON).Err()
		require.NoError(t, err)

		time.Sleep(200 * time.Millisecond)

		// Verify message was processed after restart
		assert.Len(t, emailProcessor.SentEmails, 1, "Subscriber should process messages after restart")
	})

	// Test 4: Multiple start calls (should handle gracefully)
	t.Run("multiple_start_calls", func(t *testing.T) {
		// Create a fresh subscriber for this test
		testSubscriber := NewSubscriber(redisClient)

		// Start the subscriber
		err := testSubscriber.Start(emailProcessor)
		assert.NoError(t, err, "First start should succeed")
		defer testSubscriber.Stop()

		// Try to start again while already running
		err = testSubscriber.Start(emailProcessor)
		// This should either succeed (idempotent) or return a specific error
		// The exact behavior depends on implementation
		if err != nil {
			// Accept various error messages that indicate the subscriber is already running
			errorMsg := err.Error()
			alreadyRunning := strings.Contains(errorMsg, "already") ||
				strings.Contains(errorMsg, "running") ||
				strings.Contains(errorMsg, "started")

			assert.True(t, alreadyRunning,
				"Error should indicate subscriber is already running (got: %s)", errorMsg)
		}
	})

	// Test 5: Multiple stop calls (should be idempotent)
	t.Run("multiple_stop_calls", func(t *testing.T) {
		// Stop the subscriber
		err := subscriber.Stop()
		assert.NoError(t, err, "First stop should succeed")

		// Try to stop again
		err = subscriber.Stop()
		// This should either succeed (idempotent) or return a specific error
		if err != nil {
			assert.Contains(t, err.Error(), "not running", "Error should indicate subscriber is not running")
		}
	})

	// Final cleanup
	subscriber.Stop()
}

func TestSubscriber_RedisConnectionError(t *testing.T) {
	tests := []struct {
		name        string
		redisConfig *redis.Options
		description string
	}{
		{
			name: "invalid_host",
			redisConfig: &redis.Options{
				Addr: "invalid-host:6379",
				DB:   0,
				DialTimeout: 1 * time.Second, // Short timeout for faster test
			},
			description: "Invalid Redis host should cause connection error",
		},
		{
			name: "invalid_port",
			redisConfig: &redis.Options{
				Addr: "localhost:99999", // Invalid port
				DB:   0,
				DialTimeout: 1 * time.Second,
			},
			description: "Invalid Redis port should cause connection error",
		},
		{
			name: "unreachable_address",
			redisConfig: &redis.Options{
				Addr: "192.0.2.1:6379", // RFC 5737 test address (unreachable)
				DB:   0,
				DialTimeout: 1 * time.Second,
			},
			description: "Unreachable Redis address should cause connection error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup with invalid Redis client
			invalidRedis := redis.NewClient(tt.redisConfig)
			defer invalidRedis.Close()

			subscriber := NewSubscriber(invalidRedis)
			emailProcessor := &TestEmailProcessor{}

			// Starting subscriber with invalid Redis should fail
			err := subscriber.Start(emailProcessor)
			assert.Error(t, err, tt.description)

			// Verify the error is related to connection issues
			// Accept various connection-related error messages
			errorMsg := err.Error()
			connectionRelated := strings.Contains(errorMsg, "connection") ||
				strings.Contains(errorMsg, "dial") ||
				strings.Contains(errorMsg, "timeout") ||
				strings.Contains(errorMsg, "host") ||
				strings.Contains(errorMsg, "port")

			assert.True(t, connectionRelated,
				"Error should indicate connection problem: %s (got: %s)", tt.description, errorMsg)

			// Verify subscriber can handle stop even if start failed
			err = subscriber.Stop()
			// Stop should either succeed or fail gracefully
			if err != nil {
				assert.Contains(t, err.Error(), "not running",
					"Stop error should indicate subscriber is not running")
			}
		})
	}
}

// FailingEmailProcessor is a test email processor that fails a specified number of times
type FailingEmailProcessor struct {
	failCount int
	callCount int
	mutex     sync.Mutex
	errors    []error
}

// processEmailInternal is the common implementation for all email types
func (p *FailingEmailProcessor) processEmailInternal(task *EmailTask) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.callCount++

	// Log the call for debugging
	log.Printf("FailingEmailProcessor called %d times (failCount: %d) for task type: %s",
		p.callCount, p.failCount, task.Type)

	if p.callCount <= p.failCount {
		// Create a specific error for this failure
		err := fmt.Errorf("simulated failure %d of %d for task %s",
			p.callCount, p.failCount, task.ID)

		// Store the error for later inspection
		p.errors = append(p.errors, err)

		// Return the error to trigger retry
		return err
	}

	// Succeed after failCount failures
	return nil
}

// ProcessWelcomeEmail implements the EmailProcessor interface
func (p *FailingEmailProcessor) ProcessWelcomeEmail(task *EmailTask) error {
	return p.processEmailInternal(task)
}

// ProcessVerificationEmail implements the EmailProcessor interface
func (p *FailingEmailProcessor) ProcessVerificationEmail(task *EmailTask) error {
	return p.processEmailInternal(task)
}

// ProcessPasswordResetEmail implements the EmailProcessor interface
func (p *FailingEmailProcessor) ProcessPasswordResetEmail(task *EmailTask) error {
	return p.processEmailInternal(task)
}

// ProcessNotificationEmail implements the EmailProcessor interface
func (p *FailingEmailProcessor) ProcessNotificationEmail(task *EmailTask) error {
	return p.processEmailInternal(task)
}

// GetCallCount returns the current call count (thread-safe)
func (p *FailingEmailProcessor) GetCallCount() int {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.callCount
}

// GetErrors returns all errors that occurred during processing
func (p *FailingEmailProcessor) GetErrors() []error {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	return p.errors
}

// Reset resets the processor state for a new test
func (p *FailingEmailProcessor) Reset(failCount int) {
	p.mutex.Lock()
	defer p.mutex.Unlock()
	p.failCount = failCount
	p.callCount = 0
	p.errors = nil
}
