package repository

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/systemsim/auth-service/internal/models"
)

// UserRepository handles user data operations
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository creates a new user repository
func NewUserRepository(db *sql.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create creates a new user
func (r *UserRepository) Create(user *models.User) error {
	query := `
		INSERT INTO users (
			id, email, password_hash, first_name, last_name, company,
			email_verification_token, email_verification_expires_at, email_verification_attempts,
			simulation_preferences, ui_preferences
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at`

	err := r.db.QueryRow(
		query,
		user.ID,
		user.Email,
		user.PasswordHash,
		user.FirstName,
		user.LastName,
		user.Company,
		user.EmailVerificationToken,
		user.EmailVerificationExpiresAt,
		user.EmailVerificationAttempts,
		user.SimulationPreferences,
		user.UIPreferences,
	).Scan(&user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create user: %w", err)
	}

	return nil
}

// GetByID retrieves a user by ID
func (r *UserRepository) GetByID(id uuid.UUID) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, password_hash, first_name, last_name, company,
			   email_verified, email_verification_token, email_verification_expires_at, email_verification_attempts,
			   password_reset_token, password_reset_expires_at, password_reset_attempts,
			   failed_login_attempts, locked_until, last_login_at, last_login_ip,
			   is_active, is_admin, simulation_preferences, ui_preferences,
			   created_at, updated_at, deleted_at
		FROM users 
		WHERE id = $1 AND deleted_at IS NULL`

	err := r.db.QueryRow(query, id).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Company,
		&user.EmailVerified,
		&user.EmailVerificationToken,
		&user.EmailVerificationExpiresAt,
		&user.EmailVerificationAttempts,
		&user.PasswordResetToken,
		&user.PasswordResetExpiresAt,
		&user.PasswordResetAttempts,
		&user.FailedLoginAttempts,
		&user.LockedUntil,
		&user.LastLoginAt,
		&user.LastLoginIP,
		&user.IsActive,
		&user.IsAdmin,
		&user.SimulationPreferences,
		&user.UIPreferences,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetByEmail retrieves a user by email
func (r *UserRepository) GetByEmail(email string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, password_hash, first_name, last_name, company,
			   email_verified, email_verification_token, email_verification_expires_at,
			   password_reset_token, password_reset_expires_at, password_reset_attempts,
			   failed_login_attempts, locked_until, last_login_at, last_login_ip,
			   is_active, is_admin, simulation_preferences, ui_preferences,
			   created_at, updated_at, deleted_at
		FROM users 
		WHERE email = $1 AND deleted_at IS NULL`

	err := r.db.QueryRow(query, email).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Company,
		&user.EmailVerified,
		&user.EmailVerificationToken,
		&user.EmailVerificationExpiresAt,
		&user.PasswordResetToken,
		&user.PasswordResetExpiresAt,
		&user.PasswordResetAttempts,
		&user.FailedLoginAttempts,
		&user.LockedUntil,
		&user.LastLoginAt,
		&user.LastLoginIP,
		&user.IsActive,
		&user.IsAdmin,
		&user.SimulationPreferences,
		&user.UIPreferences,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}

// GetByPasswordResetToken retrieves a user by password reset token
func (r *UserRepository) GetByPasswordResetToken(token string) (*models.User, error) {
	query := `
		SELECT id, email, first_name, last_name, company, password_hash,
		       email_verified, email_verification_token, email_verification_expires_at,
		       password_reset_token, password_reset_expires_at, password_reset_attempts,
		       failed_login_attempts, locked_until, last_login_at, last_login_ip,
		       is_active, is_admin, simulation_preferences, ui_preferences,
		       created_at, updated_at
		FROM users
		WHERE password_reset_token = $1 AND is_active = true
	`

	user := &models.User{}
	err := r.db.QueryRow(query, token).Scan(
		&user.ID, &user.Email, &user.FirstName, &user.LastName, &user.Company,
		&user.PasswordHash, &user.EmailVerified, &user.EmailVerificationToken,
		&user.EmailVerificationExpiresAt, &user.PasswordResetToken,
		&user.PasswordResetExpiresAt, &user.PasswordResetAttempts,
		&user.FailedLoginAttempts, &user.LockedUntil, &user.LastLoginAt,
		&user.LastLoginIP, &user.IsActive, &user.IsAdmin,
		&user.SimulationPreferences, &user.UIPreferences,
		&user.CreatedAt, &user.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user by reset token: %w", err)
	}

	return user, nil
}

// Update updates a user
func (r *UserRepository) Update(user *models.User) error {
	query := `
		UPDATE users SET
			first_name = $2,
			last_name = $3,
			company = $4,
			email_verified = $5,
			email_verification_token = $6,
			email_verification_expires_at = $7,
			password_reset_token = $8,
			password_reset_expires_at = $9,
			password_reset_attempts = $10,
			failed_login_attempts = $11,
			locked_until = $12,
			last_login_at = $13,
			last_login_ip = $14,
			is_active = $15,
			simulation_preferences = $16,
			ui_preferences = $17,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL
		RETURNING updated_at`

	err := r.db.QueryRow(
		query,
		user.ID,
		user.FirstName,
		user.LastName,
		user.Company,
		user.EmailVerified,
		user.EmailVerificationToken,
		user.EmailVerificationExpiresAt,
		user.PasswordResetToken,
		user.PasswordResetExpiresAt,
		user.PasswordResetAttempts,
		user.FailedLoginAttempts,
		user.LockedUntil,
		user.LastLoginAt,
		user.LastLoginIP,
		user.IsActive,
		user.SimulationPreferences,
		user.UIPreferences,
	).Scan(&user.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to update user: %w", err)
	}

	return nil
}

// UpdatePassword updates a user's password
func (r *UserRepository) UpdatePassword(userID uuid.UUID, passwordHash string) error {
	query := `
		UPDATE users SET
			password_hash = $2,
			password_reset_token = NULL,
			password_reset_expires_at = NULL,
			password_reset_attempts = 0,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.Exec(query, userID, passwordHash)
	if err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// UpdateLoginAttempts updates failed login attempts
func (r *UserRepository) UpdateLoginAttempts(userID uuid.UUID, attempts int, lockUntil *time.Time) error {
	query := `
		UPDATE users SET
			failed_login_attempts = $2,
			locked_until = $3,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.db.Exec(query, userID, attempts, lockUntil)
	if err != nil {
		return fmt.Errorf("failed to update login attempts: %w", err)
	}

	return nil
}

// UpdateLastLogin updates the last login timestamp and IP
func (r *UserRepository) UpdateLastLogin(userID uuid.UUID, ip string) error {
	query := `
		UPDATE users SET
			last_login_at = NOW(),
			last_login_ip = $2,
			failed_login_attempts = 0,
			locked_until = NULL,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	_, err := r.db.Exec(query, userID, ip)
	if err != nil {
		return fmt.Errorf("failed to update last login: %w", err)
	}

	return nil
}

// SoftDelete soft deletes a user
func (r *UserRepository) SoftDelete(userID uuid.UUID) error {
	query := `
		UPDATE users SET
			deleted_at = NOW(),
			is_active = false,
			updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`

	result, err := r.db.Exec(query, userID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}

// EmailExists checks if an email already exists
func (r *UserRepository) EmailExists(email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1 AND deleted_at IS NULL)`

	err := r.db.QueryRow(query, email).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check email existence: %w", err)
	}

	return exists, nil
}

// GetByEmailVerificationToken retrieves a user by email verification token
func (r *UserRepository) GetByEmailVerificationToken(token string) (*models.User, error) {
	user := &models.User{}
	query := `
		SELECT id, email, password_hash, first_name, last_name, company,
			   email_verified, email_verification_token, email_verification_expires_at,
			   password_reset_token, password_reset_expires_at, password_reset_attempts,
			   failed_login_attempts, locked_until, last_login_at, last_login_ip,
			   is_active, is_admin, simulation_preferences, ui_preferences,
			   created_at, updated_at, deleted_at
		FROM users
		WHERE email_verification_token = $1 AND deleted_at IS NULL`

	err := r.db.QueryRow(query, token).Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.FirstName,
		&user.LastName,
		&user.Company,
		&user.EmailVerified,
		&user.EmailVerificationToken,
		&user.EmailVerificationExpiresAt,
		&user.PasswordResetToken,
		&user.PasswordResetExpiresAt,
		&user.PasswordResetAttempts,
		&user.FailedLoginAttempts,
		&user.LockedUntil,
		&user.LastLoginAt,
		&user.LastLoginIP,
		&user.IsActive,
		&user.IsAdmin,
		&user.SimulationPreferences,
		&user.UIPreferences,
		&user.CreatedAt,
		&user.UpdatedAt,
		&user.DeletedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	return user, nil
}
