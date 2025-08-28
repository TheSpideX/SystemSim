package services

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/systemsim/auth-service/internal/models"
	"github.com/systemsim/auth-service/internal/repository"
	"github.com/systemsim/auth-service/internal/security"
)

// UserService handles user-related business logic
type UserService struct {
	userRepo    *repository.UserRepository
	sessionRepo *repository.SessionRepository
}

// NewUserService creates a new user service
func NewUserService(userRepo *repository.UserRepository, sessionRepo *repository.SessionRepository) *UserService {
	return &UserService{
		userRepo:    userRepo,
		sessionRepo: sessionRepo,
	}
}

// GetProfile retrieves a user's profile
func (s *UserService) GetProfile(userID uuid.UUID) (*models.UserResponse, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	return user.ToResponse(), nil
}

// UpdateProfile updates a user's profile
func (s *UserService) UpdateProfile(userID uuid.UUID, req *models.UpdateProfileRequest) (*models.UserResponse, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	// Update profile fields
	user.FirstName = req.FirstName
	user.LastName = req.LastName
	user.Company = req.Company

	// Update preferences if provided
	if req.SimulationPreferences != nil {
		user.SimulationPreferences = req.SimulationPreferences
	}
	if req.UIPreferences != nil {
		user.UIPreferences = req.UIPreferences
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, fmt.Errorf("failed to update profile: %w", err)
	}

	return user.ToResponse(), nil
}

// ChangePassword changes a user's password
func (s *UserService) ChangePassword(userID uuid.UUID, req *models.ChangePasswordRequest) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	// Verify current password
	if err := security.VerifyPassword(req.CurrentPassword, user.PasswordHash); err != nil {
		return fmt.Errorf("current password is incorrect")
	}

	// Hash new password
	newPasswordHash, err := security.HashPassword(req.NewPassword)
	if err != nil {
		return fmt.Errorf("failed to hash new password: %w", err)
	}

	// Update password
	if err := s.userRepo.UpdatePassword(userID, newPasswordHash); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	return nil
}

// DeleteAccount soft deletes a user account
func (s *UserService) DeleteAccount(userID uuid.UUID) error {
	if err := s.userRepo.SoftDelete(userID); err != nil {
		return fmt.Errorf("failed to delete account: %w", err)
	}

	return nil
}

// GetUserSessions retrieves all active sessions for a user
func (s *UserService) GetUserSessions(userID uuid.UUID) ([]*models.SessionResponse, error) {
	sessions, err := s.sessionRepo.GetUserSessions(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}

	// Convert to response format
	responses := make([]*models.SessionResponse, len(sessions))
	for i, session := range sessions {
		responses[i] = session.ToResponse()
	}

	return responses, nil
}

// RevokeSession revokes a specific user session
func (s *UserService) RevokeSession(userID, sessionID uuid.UUID) error {
	// First verify the session belongs to the user
	session, err := s.sessionRepo.GetByID(sessionID)
	if err != nil {
		return fmt.Errorf("session not found: %w", err)
	}

	if session.UserID != userID {
		return fmt.Errorf("session does not belong to user")
	}

	// Revoke the session
	if err := s.sessionRepo.RevokeSession(sessionID, "user_revoked"); err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	return nil
}

// RevokeAllSessions revokes all sessions for a user except the current one
func (s *UserService) RevokeAllSessions(userID, currentSessionID uuid.UUID) error {
	// Get all user sessions
	sessions, err := s.sessionRepo.GetUserSessions(userID)
	if err != nil {
		return fmt.Errorf("failed to get user sessions: %w", err)
	}

	// Revoke all sessions except the current one
	for _, session := range sessions {
		if session.ID != currentSessionID {
			if err := s.sessionRepo.RevokeSession(session.ID, "user_revoked_all"); err != nil {
				// Log error but continue with other sessions
				fmt.Printf("Failed to revoke session %s: %v\n", session.ID, err)
			}
		}
	}

	return nil
}

// UpdatePreferences updates user preferences
func (s *UserService) UpdatePreferences(userID uuid.UUID, simulationPrefs, uiPrefs models.JSONMap) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return fmt.Errorf("failed to get user: %w", err)
	}

	if simulationPrefs != nil {
		user.SimulationPreferences = simulationPrefs
	}
	if uiPrefs != nil {
		user.UIPreferences = uiPrefs
	}

	if err := s.userRepo.Update(user); err != nil {
		return fmt.Errorf("failed to update preferences: %w", err)
	}

	return nil
}

// GetPreferences retrieves user preferences
func (s *UserService) GetPreferences(userID uuid.UUID) (models.JSONMap, models.JSONMap, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user.SimulationPreferences, user.UIPreferences, nil
}

// IsEmailAvailable checks if an email is available for registration
func (s *UserService) IsEmailAvailable(email string) (bool, error) {
	exists, err := s.userRepo.EmailExists(email)
	if err != nil {
		return false, fmt.Errorf("failed to check email availability: %w", err)
	}

	return !exists, nil
}

// GetUserStats returns basic user statistics
func (s *UserService) GetUserStats(userID uuid.UUID) (map[string]interface{}, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	stats := map[string]interface{}{
		"account_created":    user.CreatedAt,
		"last_login":         user.LastLoginAt,
		"email_verified":     user.EmailVerified,
		"failed_attempts":    user.FailedLoginAttempts,
		"is_locked":          user.IsLocked(),
		"active_sessions":    0, // Would need session count
		"total_simulations":  0, // Would need simulation count from other service
	}

	return stats, nil
}

// GetUserByID returns a user by ID string (for gRPC)
func (s *UserService) GetUserByID(userID string) (*models.User, error) {
	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, fmt.Errorf("invalid user ID: %w", err)
	}

	return s.userRepo.GetByID(userUUID)
}

// GetSessionByID returns a session by ID string (for gRPC)
func (s *UserService) GetSessionByID(sessionID string) (*models.Session, error) {
	sessionUUID, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, fmt.Errorf("invalid session ID: %w", err)
	}

	return s.sessionRepo.GetByID(sessionUUID)
}
