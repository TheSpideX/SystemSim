package events

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
)

// Publisher handles publishing events to Redis
type Publisher struct {
	redis *redis.Client
	ctx   context.Context
}

// NewPublisher creates a new event publisher
func NewPublisher(redisClient *redis.Client) *Publisher {
	return &Publisher{
		redis: redisClient,
		ctx:   context.Background(),
	}
}

// Redis channel constants
const (
	// Event channels (Pub/Sub)
	ChannelAuthLogin              = "auth:events:login"
	ChannelAuthLogout             = "auth:events:logout"
	ChannelAuthRegister           = "auth:events:register"
	ChannelAuthPasswordChanged    = "auth:events:password_changed"
	ChannelAuthEmailVerified      = "auth:events:email_verified"
	ChannelAuthAccountLocked      = "auth:events:account_locked"
	ChannelAuthPermissionChanged  = "auth:events:permission_changed"
	ChannelAuthSessionCreated     = "auth:events:session_created"
	ChannelAuthSessionRevoked     = "auth:events:session_revoked"
	
	// Background task channels (Pub/Sub)
	ChannelAsyncEmailQueue        = "async:email:queue"
	ChannelAsyncNotificationQueue = "async:notification:queue"
	
	// System channels (Pub/Sub)
	ChannelSystemAnnouncements    = "system:announcements"
)

// PublishLoginEvent publishes a login event
func (p *Publisher) PublishLoginEvent(event *LoginEvent) error {
	return p.publishEvent(ChannelAuthLogin, event)
}

// PublishLogoutEvent publishes a logout event
func (p *Publisher) PublishLogoutEvent(event *LogoutEvent) error {
	return p.publishEvent(ChannelAuthLogout, event)
}

// PublishRegisterEvent publishes a registration event
func (p *Publisher) PublishRegisterEvent(event *RegisterEvent) error {
	return p.publishEvent(ChannelAuthRegister, event)
}

// PublishPermissionChangedEvent publishes a permission changed event
func (p *Publisher) PublishPermissionChangedEvent(event *PermissionChangedEvent) error {
	return p.publishEvent(ChannelAuthPermissionChanged, event)
}

// PublishSessionEvent publishes a session event
func (p *Publisher) PublishSessionEvent(event *SessionEvent) error {
	var channel string
	switch event.Action {
	case "created":
		channel = ChannelAuthSessionCreated
	case "revoked", "expired":
		channel = ChannelAuthSessionRevoked
	default:
		return fmt.Errorf("unknown session action: %s", event.Action)
	}
	return p.publishEvent(channel, event)
}

// PublishEmailTask publishes an email task to the background queue
func (p *Publisher) PublishEmailTask(task *EmailTask) error {
	return p.publishEvent(ChannelAsyncEmailQueue, task)
}

// PublishSystemAnnouncement publishes a system announcement
func (p *Publisher) PublishSystemAnnouncement(announcement *SystemAnnouncement) error {
	return p.publishEvent(ChannelSystemAnnouncements, announcement)
}

// publishEvent is a generic method to publish any event to a Redis channel
func (p *Publisher) publishEvent(channel string, event interface{}) error {
	// Serialize event to JSON
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	// Publish to Redis channel
	err = p.redis.Publish(p.ctx, channel, data).Err()
	if err != nil {
		return fmt.Errorf("failed to publish event to channel %s: %w", channel, err)
	}

	log.Printf("Published event to channel %s: %s", channel, string(data))
	return nil
}

// PublishLoginSuccess publishes a successful login event
func (p *Publisher) PublishLoginSuccess(userID, sessionID, email, ipAddress, userAgent string) error {
	event := NewLoginEvent(userID, sessionID, email, ipAddress, userAgent, true, "")
	return p.PublishLoginEvent(event)
}

// PublishLoginFailure publishes a failed login event
func (p *Publisher) PublishLoginFailure(email, ipAddress, userAgent, reason string) error {
	event := NewLoginEvent("", "", email, ipAddress, userAgent, false, reason)
	return p.PublishLoginEvent(event)
}

// PublishUserLogout publishes a user logout event
func (p *Publisher) PublishUserLogout(userID, sessionID, reason string) error {
	event := NewLogoutEvent(userID, sessionID, reason)
	return p.PublishLogoutEvent(event)
}

// PublishUserRegistration publishes a user registration event
func (p *Publisher) PublishUserRegistration(userID, email, firstName, lastName, company, ipAddress string) error {
	event := NewRegisterEvent(userID, email, firstName, lastName, company, ipAddress)
	return p.PublishRegisterEvent(event)
}

// PublishPermissionUpdate publishes a permission change event
func (p *Publisher) PublishPermissionUpdate(userID, changedBy, action string, permissions, roles []string) error {
	event := NewPermissionChangedEvent(userID, changedBy, action, permissions, roles)
	return p.PublishPermissionChangedEvent(event)
}

// PublishSessionCreated publishes a session created event
func (p *Publisher) PublishSessionCreated(userID, sessionID, ipAddress, userAgent string) error {
	event := NewSessionEvent(EventTypeSessionCreated, userID, sessionID, "created", ipAddress, userAgent, "")
	return p.PublishSessionEvent(event)
}

// PublishSessionRevoked publishes a session revoked event
func (p *Publisher) PublishSessionRevoked(userID, sessionID, reason string) error {
	event := NewSessionEvent(EventTypeSessionRevoked, userID, sessionID, "revoked", "", "", reason)
	return p.PublishSessionEvent(event)
}

// Email task helpers
func (p *Publisher) PublishWelcomeEmail(to, firstName string) error {
	task := NewEmailTask(
		"welcome",
		to,
		"Welcome to SystemSim!",
		"welcome",
		map[string]string{
			"first_name": firstName,
		},
		2, // normal priority
	)
	return p.PublishEmailTask(task)
}

func (p *Publisher) PublishVerificationEmail(to, firstName, verificationToken string) error {
	task := NewEmailTask(
		"verification",
		to,
		"Verify your email address",
		"email_verification",
		map[string]string{
			"first_name":         firstName,
			"verification_token": verificationToken,
		},
		1, // high priority
	)
	return p.PublishEmailTask(task)
}

func (p *Publisher) PublishPasswordResetEmail(to, firstName, resetToken string) error {
	task := NewEmailTask(
		"password_reset",
		to,
		"Reset your password",
		"password_reset",
		map[string]string{
			"first_name":  firstName,
			"reset_token": resetToken,
		},
		1, // high priority
	)
	return p.PublishEmailTask(task)
}

// Health check for publisher
func (p *Publisher) HealthCheck() error {
	ctx, cancel := context.WithTimeout(p.ctx, 5*time.Second)
	defer cancel()
	
	return p.redis.Ping(ctx).Err()
}
