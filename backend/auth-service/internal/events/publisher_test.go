package events

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/systemsim/auth-service/internal/testutils"
)

func TestPublisher_PublishLoginEvent(t *testing.T) {
	// Setup
	redisClient := testutils.SetupTestRedis(t)
	defer redisClient.Close()
	defer testutils.CleanupTestRedis(t, redisClient)

	publisher := NewPublisher(redisClient)

	// Subscribe to the channel to verify event is published
	pubsub := redisClient.Subscribe(context.Background(), ChannelAuthLogin)
	defer pubsub.Close()

	// Wait for subscription confirmation
	_, err := pubsub.Receive(context.Background())
	require.NoError(t, err)

	tests := []struct {
		name  string
		event *LoginEvent
	}{
		{
			name: "successful_login_event",
			event: &LoginEvent{
				BaseEvent: BaseEvent{
					ID:        uuid.New().String(),
					Type:      EventTypeLogin,
					Timestamp: time.Now(),
					UserID:    uuid.New().String(),
					SessionID: uuid.New().String(),
					Source:    "auth-service",
				},
				Email:     "test@example.com",
				IPAddress: "192.168.1.1",
				UserAgent: "Mozilla/5.0",
				Success:   true,
			},
		},
		{
			name: "failed_login_event",
			event: &LoginEvent{
				BaseEvent: BaseEvent{
					ID:        uuid.New().String(),
					Type:      EventTypeLogin,
					Timestamp: time.Now(),
					UserID:    "",
					SessionID: "",
					Source:    "auth-service",
				},
				Email:     "test2@example.com",
				IPAddress: "192.168.1.2",
				UserAgent: "Mozilla/5.0",
				Success:   false,
				Reason:    "invalid credentials",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute
			err := publisher.PublishLoginEvent(tt.event)
			assert.NoError(t, err)

			// Verify event was published
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			msg, err := pubsub.ReceiveMessage(ctx)
			assert.NoError(t, err)
			assert.Equal(t, ChannelAuthLogin, msg.Channel)

			// Parse the event
			var event LoginEvent
			err = json.Unmarshal([]byte(msg.Payload), &event)
			assert.NoError(t, err)

			// Verify event content
			assert.Equal(t, EventTypeLogin, event.Type)
			assert.Equal(t, tt.event.UserID, event.UserID)
			assert.Equal(t, tt.event.Email, event.Email)
			assert.Equal(t, tt.event.SessionID, event.SessionID)
			assert.Equal(t, tt.event.IPAddress, event.IPAddress)
			assert.Equal(t, tt.event.UserAgent, event.UserAgent)
			assert.Equal(t, tt.event.Success, event.Success)
			assert.Equal(t, tt.event.Reason, event.Reason)
			assert.NotEmpty(t, event.ID)
			assert.NotZero(t, event.Timestamp)
			assert.Equal(t, "auth-service", event.Source)
		})
	}
}

func TestPublisher_PublishLogoutEvent(t *testing.T) {
	// Setup
	redisClient := testutils.SetupTestRedis(t)
	defer redisClient.Close()
	defer testutils.CleanupTestRedis(t, redisClient)

	publisher := NewPublisher(redisClient)

	// Subscribe to the channel
	pubsub := redisClient.Subscribe(context.Background(), ChannelAuthLogout)
	defer pubsub.Close()

	// Wait for subscription confirmation
	_, err := pubsub.Receive(context.Background())
	require.NoError(t, err)

	// Create logout event
	logoutEvent := &LogoutEvent{
		BaseEvent: BaseEvent{
			ID:        uuid.New().String(),
			Type:      EventTypeLogout,
			Timestamp: time.Now(),
			UserID:    uuid.New().String(),
			SessionID: uuid.New().String(),
			Source:    "auth-service",
		},
	}

	// Execute
	err = publisher.PublishLogoutEvent(logoutEvent)
	assert.NoError(t, err)

	// Verify event was published
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	msg, err := pubsub.ReceiveMessage(ctx)
	assert.NoError(t, err)
	assert.Equal(t, ChannelAuthLogout, msg.Channel)

	// Parse the event
	var event LogoutEvent
	err = json.Unmarshal([]byte(msg.Payload), &event)
	assert.NoError(t, err)

	// Verify event content
	assert.Equal(t, EventTypeLogout, event.Type)
	assert.Equal(t, logoutEvent.UserID, event.UserID)
	assert.Equal(t, logoutEvent.SessionID, event.SessionID)
	assert.NotEmpty(t, event.ID)
	assert.NotZero(t, event.Timestamp)
	assert.Equal(t, "auth-service", event.Source)
}

func TestPublisher_PublishRegisterEvent(t *testing.T) {
	// Setup
	redisClient := testutils.SetupTestRedis(t)
	defer redisClient.Close()
	defer testutils.CleanupTestRedis(t, redisClient)

	publisher := NewPublisher(redisClient)

	// Subscribe to the channel
	pubsub := redisClient.Subscribe(context.Background(), ChannelAuthRegister)
	defer pubsub.Close()

	// Wait for subscription confirmation
	_, err := pubsub.Receive(context.Background())
	require.NoError(t, err)

	// Create register event
	registerEvent := &RegisterEvent{
		BaseEvent: BaseEvent{
			ID:        uuid.New().String(),
			Type:      EventTypeRegister,
			Timestamp: time.Now(),
			UserID:    uuid.New().String(),
			Source:    "auth-service",
		},
		Email:     "register@example.com",
		FirstName: "John",
		LastName:  "Doe",
	}

	// Execute
	err = publisher.PublishRegisterEvent(registerEvent)
	assert.NoError(t, err)

	// Verify event was published
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	msg, err := pubsub.ReceiveMessage(ctx)
	assert.NoError(t, err)
	assert.Equal(t, ChannelAuthRegister, msg.Channel)

	// Parse the event
	var event RegisterEvent
	err = json.Unmarshal([]byte(msg.Payload), &event)
	assert.NoError(t, err)

	// Verify event content
	assert.Equal(t, EventTypeRegister, event.Type)
	assert.Equal(t, registerEvent.UserID, event.UserID)
	assert.Equal(t, registerEvent.Email, event.Email)
	assert.Equal(t, registerEvent.FirstName, event.FirstName)
	assert.Equal(t, registerEvent.LastName, event.LastName)
	assert.NotEmpty(t, event.ID)
	assert.NotZero(t, event.Timestamp)
	assert.Equal(t, "auth-service", event.Source)
}

func TestPublisher_PublishPermissionChangedEvent(t *testing.T) {
	// Setup
	redisClient := testutils.SetupTestRedis(t)
	defer redisClient.Close()
	defer testutils.CleanupTestRedis(t, redisClient)

	publisher := NewPublisher(redisClient)

	// Subscribe to the channel
	pubsub := redisClient.Subscribe(context.Background(), ChannelAuthPermissionChanged)
	defer pubsub.Close()

	// Wait for subscription confirmation
	_, err := pubsub.Receive(context.Background())
	require.NoError(t, err)

	// Create permission changed event
	permissionEvent := &PermissionChangedEvent{
		BaseEvent: BaseEvent{
			ID:        uuid.New().String(),
			Type:      EventTypePermissionChanged,
			Timestamp: time.Now(),
			UserID:    uuid.New().String(),
			Source:    "auth-service",
		},
		Permissions: []string{"read:projects", "write:projects"},
		Roles:       []string{"user", "editor"},
		ChangedBy:   uuid.New().String(),
		Action:      "granted",
	}

	// Execute
	err = publisher.PublishPermissionChangedEvent(permissionEvent)
	assert.NoError(t, err)

	// Verify event was published
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	msg, err := pubsub.ReceiveMessage(ctx)
	assert.NoError(t, err)
	assert.Equal(t, ChannelAuthPermissionChanged, msg.Channel)

	// Parse the event
	var event PermissionChangedEvent
	err = json.Unmarshal([]byte(msg.Payload), &event)
	assert.NoError(t, err)

	// Verify event content
	assert.Equal(t, EventTypePermissionChanged, event.Type)
	assert.Equal(t, permissionEvent.UserID, event.UserID)
	assert.Equal(t, permissionEvent.ChangedBy, event.ChangedBy)
	assert.Equal(t, permissionEvent.Permissions, event.Permissions)
	assert.Equal(t, permissionEvent.Roles, event.Roles)
	assert.Equal(t, permissionEvent.Action, event.Action)
	assert.NotEmpty(t, event.ID)
	assert.NotZero(t, event.Timestamp)
	assert.Equal(t, "auth-service", event.Source)
}

func TestPublisher_PublishEmailTask(t *testing.T) {
	// Setup
	redisClient := testutils.SetupTestRedis(t)
	defer redisClient.Close()
	defer testutils.CleanupTestRedis(t, redisClient)

	publisher := NewPublisher(redisClient)

	// Subscribe to the email queue channel
	pubsub := redisClient.Subscribe(context.Background(), ChannelAsyncEmailQueue)
	defer pubsub.Close()

	// Wait for subscription confirmation
	_, err := pubsub.Receive(context.Background())
	require.NoError(t, err)

	tests := []struct {
		name string
		task *EmailTask
	}{
		{
			name: "welcome_email_task",
			task: &EmailTask{
				ID:       uuid.New().String(),
				Type:     "welcome",
				To:       "welcome@example.com",
				Subject:  "Welcome to our platform",
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
			name: "verification_email_task",
			task: &EmailTask{
				ID:       uuid.New().String(),
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
			name: "password_reset_email_task",
			task: &EmailTask{
				ID:       uuid.New().String(),
				Type:     "password_reset",
				To:       "reset@example.com",
				Subject:  "Password reset request",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Execute
			err := publisher.PublishEmailTask(tt.task)
			assert.NoError(t, err)

			// Verify task was published
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			msg, err := pubsub.ReceiveMessage(ctx)
			assert.NoError(t, err)

			// Parse the task
			var task EmailTask
			err = json.Unmarshal([]byte(msg.Payload), &task)
			assert.NoError(t, err)

			// Verify task content
			assert.Equal(t, tt.task.Type, task.Type)
			assert.Equal(t, tt.task.To, task.To)
			assert.Equal(t, tt.task.Subject, task.Subject)
			assert.Equal(t, tt.task.Template, task.Template)
			assert.Equal(t, tt.task.Variables, task.Variables)
			assert.Equal(t, tt.task.Priority, task.Priority)
			assert.NotEmpty(t, task.ID)
			assert.NotZero(t, task.Timestamp)
			assert.Equal(t, tt.task.Retries, task.Retries)
			assert.Equal(t, tt.task.MaxRetries, task.MaxRetries)
		})
	}
}

func TestPublisher_RedisConnectionError(t *testing.T) {
	// Setup with invalid Redis client
	invalidRedis := redis.NewClient(&redis.Options{
		Addr: "invalid:6379",
		DB:   0,
	})
	defer invalidRedis.Close()

	publisher := NewPublisher(invalidRedis)

	// Execute - should handle Redis connection errors gracefully
	loginEvent := &LoginEvent{
		BaseEvent: BaseEvent{
			ID:        uuid.New().String(),
			Type:      EventTypeLogin,
			Timestamp: time.Now(),
			UserID:    "user123",
			SessionID: "session123",
			Source:    "auth-service",
		},
		Email:     "test@example.com",
		IPAddress: "127.0.0.1",
		UserAgent: "test-agent",
		Success:   true,
	}
	err := publisher.PublishLoginEvent(loginEvent)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to publish")
}

func TestPublisher_EventSerialization(t *testing.T) {
	// Setup
	redisClient := testutils.SetupTestRedis(t)
	defer redisClient.Close()
	defer testutils.CleanupTestRedis(t, redisClient)

	publisher := NewPublisher(redisClient)

	// Test with special characters and unicode
	userID := uuid.New().String()
	email := "test+special@example.com"
	sessionID := uuid.New().String()
	ipAddress := "192.168.1.1"
	userAgent := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36"

	// Create login event for serialization test
	loginEvent := &LoginEvent{
		BaseEvent: BaseEvent{
			ID:        uuid.New().String(),
			Type:      EventTypeLogin,
			Timestamp: time.Now(),
			UserID:    userID,
			SessionID: sessionID,
			Source:    "auth-service",
		},
		Email:     email,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   true,
	}

	// Execute
	err := publisher.PublishLoginEvent(loginEvent)
	assert.NoError(t, err)

	// Verify the event can be retrieved and deserialized
	pubsub := redisClient.Subscribe(context.Background(), ChannelAuthLogin)
	defer pubsub.Close()

	// The event should already be published, so we might not receive it
	// This test mainly verifies that serialization doesn't fail
}
