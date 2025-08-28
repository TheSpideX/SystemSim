package models

import (
	"time"

	"github.com/google/uuid"
)

// Session represents a user session
type Session struct {
	ID                 uuid.UUID `json:"id" db:"id"`
	UserID             uuid.UUID `json:"user_id" db:"user_id"`
	TokenHash          string    `json:"-" db:"token_hash"`
	RefreshTokenHash   *string   `json:"-" db:"refresh_token_hash"`
	
	// Session metadata
	DeviceInfo JSONMap `json:"device_info" db:"device_info"`
	UserAgent  *string `json:"user_agent" db:"user_agent"`
	IPAddress  *string `json:"ip_address" db:"ip_address"`
	
	// Session timing
	ExpiresAt        time.Time  `json:"expires_at" db:"expires_at"`
	RefreshExpiresAt *time.Time `json:"refresh_expires_at" db:"refresh_expires_at"`
	LastUsedAt       time.Time  `json:"last_used_at" db:"last_used_at"`
	
	// Session status
	IsActive       bool       `json:"is_active" db:"is_active"`
	RevokedAt      *time.Time `json:"revoked_at" db:"revoked_at"`
	RevokedReason  *string    `json:"revoked_reason" db:"revoked_reason"`
	
	// Audit fields
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// IsExpired checks if the session is expired
func (s *Session) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsRefreshExpired checks if the refresh token is expired
func (s *Session) IsRefreshExpired() bool {
	if s.RefreshExpiresAt == nil {
		return true
	}
	return time.Now().After(*s.RefreshExpiresAt)
}

// IsValid checks if the session is valid (active, not expired, not revoked)
func (s *Session) IsValid() bool {
	return s.IsActive && !s.IsExpired() && s.RevokedAt == nil
}

// CanRefresh checks if the session can be refreshed
func (s *Session) CanRefresh() bool {
	return s.IsActive && 
		   s.RefreshTokenHash != nil && 
		   !s.IsRefreshExpired() && 
		   s.RevokedAt == nil
}

// Revoke marks the session as revoked
func (s *Session) Revoke(reason string) {
	now := time.Now()
	s.IsActive = false
	s.RevokedAt = &now
	s.RevokedReason = &reason
}

// SessionResponse represents session data returned to clients
type SessionResponse struct {
	ID           uuid.UUID `json:"id"`
	DeviceInfo   JSONMap   `json:"device_info"`
	IPAddress    *string   `json:"ip_address"`
	ExpiresAt    time.Time `json:"expires_at"`
	LastUsedAt   time.Time `json:"last_used_at"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	RememberMe   bool      `json:"remember_me"` // indicates if this is a remember me session
}

// ToResponse converts Session to SessionResponse
func (s *Session) ToResponse() *SessionResponse {
	// Determine if this is a remember me session based on duration
	// If the session expires more than 1 day from creation, it's likely a remember me session
	duration := s.ExpiresAt.Sub(s.CreatedAt)
	isRememberMe := duration > 24*time.Hour

	return &SessionResponse{
		ID:         s.ID,
		DeviceInfo: s.DeviceInfo,
		IPAddress:  s.IPAddress,
		ExpiresAt:  s.ExpiresAt,
		LastUsedAt: s.LastUsedAt,
		IsActive:   s.IsActive,
		CreatedAt:  s.CreatedAt,
		RememberMe: isRememberMe,
	}
}
