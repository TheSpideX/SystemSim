package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// User represents a user in the system
type User struct {
	ID       uuid.UUID `json:"id" db:"id"`
	Email    string    `json:"email" db:"email" validate:"required,email"`
	Password string    `json:"password,omitempty" validate:"required,min=8"`
	
	// Profile information
	FirstName string `json:"first_name" db:"first_name" validate:"max=100"`
	LastName  string `json:"last_name" db:"last_name" validate:"max=100"`
	Company   string `json:"company" db:"company" validate:"max=200"`
	
	// Security fields
	PasswordHash                string     `json:"-" db:"password_hash"`
	EmailVerified               bool       `json:"email_verified" db:"email_verified"`
	EmailVerificationToken      *string    `json:"-" db:"email_verification_token"`
	EmailVerificationExpiresAt  *time.Time `json:"-" db:"email_verification_expires_at"`
	EmailVerificationAttempts   int        `json:"-" db:"email_verification_attempts"`
	
	// Password reset
	PasswordResetToken     *string    `json:"-" db:"password_reset_token"`
	PasswordResetExpiresAt *time.Time `json:"-" db:"password_reset_expires_at"`
	PasswordResetAttempts  int        `json:"-" db:"password_reset_attempts"`
	
	// Account security
	FailedLoginAttempts int        `json:"-" db:"failed_login_attempts"`
	LockedUntil         *time.Time `json:"-" db:"locked_until"`
	LastLoginAt         *time.Time `json:"last_login_at" db:"last_login_at"`
	LastLoginIP         *string    `json:"last_login_ip" db:"last_login_ip"`
	
	// Account status
	IsActive bool `json:"is_active" db:"is_active"`
	IsAdmin  bool `json:"is_admin" db:"is_admin"`
	
	// Preferences
	SimulationPreferences JSONMap `json:"simulation_preferences" db:"simulation_preferences"`
	UIPreferences         JSONMap `json:"ui_preferences" db:"ui_preferences"`
	
	// Audit fields
	CreatedAt time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" db:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty" db:"deleted_at"`
}

// UserResponse represents user data returned to clients (without sensitive fields)
type UserResponse struct {
	ID        uuid.UUID `json:"id"`
	Email     string    `json:"email"`
	FirstName string    `json:"first_name"`
	LastName  string    `json:"last_name"`
	Company   string    `json:"company"`
	
	EmailVerified bool       `json:"email_verified"`
	LastLoginAt   *time.Time `json:"last_login_at"`
	IsActive      bool       `json:"is_active"`
	IsAdmin       bool       `json:"is_admin"`
	
	SimulationPreferences JSONMap `json:"simulation_preferences"`
	UIPreferences         JSONMap `json:"ui_preferences"`
	
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ToResponse converts User to UserResponse
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:                    u.ID,
		Email:                 u.Email,
		FirstName:             u.FirstName,
		LastName:              u.LastName,
		Company:               u.Company,
		EmailVerified:         u.EmailVerified,
		LastLoginAt:           u.LastLoginAt,
		IsActive:              u.IsActive,
		IsAdmin:               u.IsAdmin,
		SimulationPreferences: u.SimulationPreferences,
		UIPreferences:         u.UIPreferences,
		CreatedAt:             u.CreatedAt,
		UpdatedAt:             u.UpdatedAt,
	}
}

// IsLocked checks if the user account is currently locked
func (u *User) IsLocked() bool {
	return u.LockedUntil != nil && u.LockedUntil.After(time.Now())
}

// CanAttemptLogin checks if user can attempt login (not locked and active)
func (u *User) CanAttemptLogin() bool {
	return u.IsActive && !u.IsLocked()
}

// JSONMap is a custom type for JSONB fields
type JSONMap map[string]interface{}

// Value implements the driver.Valuer interface for database storage
func (j JSONMap) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements the sql.Scanner interface for database retrieval
func (j *JSONMap) Scan(value interface{}) error {
	if value == nil {
		*j = make(JSONMap)
		return nil
	}
	
	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into JSONMap", value)
	}
	
	return json.Unmarshal(bytes, j)
}
