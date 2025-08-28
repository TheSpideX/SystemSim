package events

import (
	"time"

	"github.com/google/uuid"
)

// EventType represents the type of event
type EventType string

const (
	// Auth Events
	EventTypeLogin              EventType = "login"
	EventTypeLogout             EventType = "logout"
	EventTypeRegister           EventType = "register"
	EventTypePasswordChanged    EventType = "password_changed"
	EventTypeEmailVerified      EventType = "email_verified"
	EventTypeAccountLocked      EventType = "account_locked"
	EventTypeAccountUnlocked    EventType = "account_unlocked"
	
	// RBAC Events
	EventTypePermissionChanged  EventType = "permission_changed"
	EventTypeRoleAssigned       EventType = "role_assigned"
	EventTypeRoleRemoved        EventType = "role_removed"
	
	// Session Events
	EventTypeSessionCreated     EventType = "session_created"
	EventTypeSessionRevoked     EventType = "session_revoked"
	EventTypeAllSessionsRevoked EventType = "all_sessions_revoked"
)

// BaseEvent contains common fields for all events
type BaseEvent struct {
	ID        string    `json:"id"`
	Type      EventType `json:"type"`
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"user_id,omitempty"`
	SessionID string    `json:"session_id,omitempty"`
	Source    string    `json:"source"` // "auth-service"
}

// LoginEvent represents a user login event
type LoginEvent struct {
	BaseEvent
	Email     string `json:"email"`
	IPAddress string `json:"ip_address,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	Success   bool   `json:"success"`
	Reason    string `json:"reason,omitempty"` // failure reason if success=false
}

// LogoutEvent represents a user logout event
type LogoutEvent struct {
	BaseEvent
	Reason string `json:"reason"` // "user_logout", "token_expired", "admin_revoke", "session_timeout"
}

// RegisterEvent represents a user registration event
type RegisterEvent struct {
	BaseEvent
	Email     string `json:"email"`
	FirstName string `json:"first_name,omitempty"`
	LastName  string `json:"last_name,omitempty"`
	Company   string `json:"company,omitempty"`
	IPAddress string `json:"ip_address,omitempty"`
}

// PermissionChangedEvent represents a permission change event
type PermissionChangedEvent struct {
	BaseEvent
	Permissions []string `json:"permissions"`
	Roles       []string `json:"roles"`
	ChangedBy   string   `json:"changed_by"`
	Action      string   `json:"action"` // "granted", "revoked", "updated"
}

// SessionEvent represents session-related events
type SessionEvent struct {
	BaseEvent
	Action    string `json:"action"` // "created", "revoked", "expired"
	IPAddress string `json:"ip_address,omitempty"`
	UserAgent string `json:"user_agent,omitempty"`
	Reason    string `json:"reason,omitempty"`
}

// EmailTask represents a background email task
type EmailTask struct {
	ID        string            `json:"id"`
	Type      string            `json:"type"` // "welcome", "verification", "password_reset", "notification"
	To        string            `json:"to"`
	Subject   string            `json:"subject"`
	Template  string            `json:"template"`
	Variables map[string]string `json:"variables"`
	Priority  int               `json:"priority"` // 1=high, 2=normal, 3=low
	Timestamp time.Time         `json:"timestamp"`
	Retries   int               `json:"retries"`
	MaxRetries int              `json:"max_retries"`
}

// SystemAnnouncement represents system-wide announcements
type SystemAnnouncement struct {
	ID        string    `json:"id"`
	Type      string    `json:"type"` // "maintenance", "security_alert", "feature_update"
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Severity  string    `json:"severity"` // "info", "warning", "critical"
	Source    string    `json:"source"`
}

// NewBaseEvent creates a new base event
func NewBaseEvent(eventType EventType, userID, sessionID string) BaseEvent {
	return BaseEvent{
		ID:        uuid.New().String(),
		Type:      eventType,
		Timestamp: time.Now(),
		UserID:    userID,
		SessionID: sessionID,
		Source:    "auth-service",
	}
}

// NewLoginEvent creates a new login event
func NewLoginEvent(userID, sessionID, email, ipAddress, userAgent string, success bool, reason string) *LoginEvent {
	return &LoginEvent{
		BaseEvent: NewBaseEvent(EventTypeLogin, userID, sessionID),
		Email:     email,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Success:   success,
		Reason:    reason,
	}
}

// NewLogoutEvent creates a new logout event
func NewLogoutEvent(userID, sessionID, reason string) *LogoutEvent {
	return &LogoutEvent{
		BaseEvent: NewBaseEvent(EventTypeLogout, userID, sessionID),
		Reason:    reason,
	}
}

// NewRegisterEvent creates a new registration event
func NewRegisterEvent(userID, email, firstName, lastName, company, ipAddress string) *RegisterEvent {
	return &RegisterEvent{
		BaseEvent: NewBaseEvent(EventTypeRegister, userID, ""),
		Email:     email,
		FirstName: firstName,
		LastName:  lastName,
		Company:   company,
		IPAddress: ipAddress,
	}
}

// NewPermissionChangedEvent creates a new permission changed event
func NewPermissionChangedEvent(userID, changedBy, action string, permissions, roles []string) *PermissionChangedEvent {
	return &PermissionChangedEvent{
		BaseEvent:   NewBaseEvent(EventTypePermissionChanged, userID, ""),
		Permissions: permissions,
		Roles:       roles,
		ChangedBy:   changedBy,
		Action:      action,
	}
}

// NewSessionEvent creates a new session event
func NewSessionEvent(eventType EventType, userID, sessionID, action, ipAddress, userAgent, reason string) *SessionEvent {
	return &SessionEvent{
		BaseEvent: NewBaseEvent(eventType, userID, sessionID),
		Action:    action,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Reason:    reason,
	}
}

// NewEmailTask creates a new email task
func NewEmailTask(taskType, to, subject, template string, variables map[string]string, priority int) *EmailTask {
	return &EmailTask{
		ID:         uuid.New().String(),
		Type:       taskType,
		To:         to,
		Subject:    subject,
		Template:   template,
		Variables:  variables,
		Priority:   priority,
		Timestamp:  time.Now(),
		Retries:    0,
		MaxRetries: 3,
	}
}
