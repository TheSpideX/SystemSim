package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/systemsim/auth-service/internal/models"
)

// SessionRepository handles session data operations
type SessionRepository struct {
	db    *sql.DB
	redis *redis.Client
}

// NewSessionRepository creates a new session repository
func NewSessionRepository(db *sql.DB, redis *redis.Client) *SessionRepository {
	return &SessionRepository{
		db:    db,
		redis: redis,
	}
}

// Create creates a new session
func (r *SessionRepository) Create(session *models.Session) error {
	query := `
		INSERT INTO user_sessions (
			id, user_id, token_hash, refresh_token_hash,
			device_info, user_agent, ip_address,
			expires_at, refresh_expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING created_at, updated_at`

	err := r.db.QueryRow(
		query,
		session.ID,
		session.UserID,
		session.TokenHash,
		session.RefreshTokenHash,
		session.DeviceInfo,
		session.UserAgent,
		session.IPAddress,
		session.ExpiresAt,
		session.RefreshExpiresAt,
	).Scan(&session.CreatedAt, &session.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to create session: %w", err)
	}

	// Cache session in Redis for fast lookup
	if r.redis != nil {
		r.cacheSession(session)
	}

	return nil
}

// GetByTokenHash retrieves a session by token hash
func (r *SessionRepository) GetByTokenHash(tokenHash string) (*models.Session, error) {
	// Try Redis cache first
	if r.redis != nil {
		if session, err := r.getSessionFromCache(tokenHash); err == nil {
			return session, nil
		}
	}

	// Fall back to database
	session := &models.Session{}
	query := `
		SELECT id, user_id, token_hash, refresh_token_hash,
			   device_info, user_agent, ip_address,
			   expires_at, refresh_expires_at, last_used_at,
			   is_active, revoked_at, revoked_reason,
			   created_at, updated_at
		FROM user_sessions 
		WHERE token_hash = $1`

	err := r.db.QueryRow(query, tokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&session.RefreshTokenHash,
		&session.DeviceInfo,
		&session.UserAgent,
		&session.IPAddress,
		&session.ExpiresAt,
		&session.RefreshExpiresAt,
		&session.LastUsedAt,
		&session.IsActive,
		&session.RevokedAt,
		&session.RevokedReason,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Cache in Redis for future lookups
	if r.redis != nil {
		r.cacheSession(session)
	}

	return session, nil
}

// GetByRefreshTokenHash retrieves a session by refresh token hash
func (r *SessionRepository) GetByRefreshTokenHash(refreshTokenHash string) (*models.Session, error) {
	session := &models.Session{}
	query := `
		SELECT id, user_id, token_hash, refresh_token_hash,
			   device_info, user_agent, ip_address,
			   expires_at, refresh_expires_at, last_used_at,
			   is_active, revoked_at, revoked_reason,
			   created_at, updated_at
		FROM user_sessions 
		WHERE refresh_token_hash = $1`

	err := r.db.QueryRow(query, refreshTokenHash).Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&session.RefreshTokenHash,
		&session.DeviceInfo,
		&session.UserAgent,
		&session.IPAddress,
		&session.ExpiresAt,
		&session.RefreshExpiresAt,
		&session.LastUsedAt,
		&session.IsActive,
		&session.RevokedAt,
		&session.RevokedReason,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

// Update updates a session
func (r *SessionRepository) Update(session *models.Session) error {
	query := `
		UPDATE user_sessions SET
			token_hash = $2,
			refresh_token_hash = $3,
			expires_at = $4,
			refresh_expires_at = $5,
			last_used_at = $6,
			is_active = $7,
			revoked_at = $8,
			revoked_reason = $9,
			updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`

	err := r.db.QueryRow(
		query,
		session.ID,
		session.TokenHash,
		session.RefreshTokenHash,
		session.ExpiresAt,
		session.RefreshExpiresAt,
		session.LastUsedAt,
		session.IsActive,
		session.RevokedAt,
		session.RevokedReason,
	).Scan(&session.UpdatedAt)

	if err != nil {
		return fmt.Errorf("failed to update session: %w", err)
	}

	// Update cache
	if r.redis != nil {
		r.cacheSession(session)
	}

	return nil
}

// UpdateLastUsed updates the last used timestamp
func (r *SessionRepository) UpdateLastUsed(sessionID uuid.UUID) error {
	query := `
		UPDATE user_sessions SET
			last_used_at = NOW(),
			updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.Exec(query, sessionID)
	if err != nil {
		return fmt.Errorf("failed to update last used: %w", err)
	}

	return nil
}

// RevokeSession revokes a specific session
func (r *SessionRepository) RevokeSession(sessionID uuid.UUID, reason string) error {
	query := `
		UPDATE user_sessions SET
			is_active = false,
			revoked_at = NOW(),
			revoked_reason = $2,
			updated_at = NOW()
		WHERE id = $1`

	_, err := r.db.Exec(query, sessionID, reason)
	if err != nil {
		return fmt.Errorf("failed to revoke session: %w", err)
	}

	// Remove from cache
	if r.redis != nil {
		r.removeSessionFromCache(sessionID.String())
	}

	return nil
}

// RevokeAllUserSessions revokes all sessions for a user
func (r *SessionRepository) RevokeAllUserSessions(userID uuid.UUID, reason string) error {
	query := `
		UPDATE user_sessions SET
			is_active = false,
			revoked_at = NOW(),
			revoked_reason = $2,
			updated_at = NOW()
		WHERE user_id = $1 AND is_active = true`

	_, err := r.db.Exec(query, userID, reason)
	if err != nil {
		return fmt.Errorf("failed to revoke user sessions: %w", err)
	}

	// Clear user sessions from cache
	if r.redis != nil {
		r.clearUserSessionsFromCache(userID)
	}

	return nil
}

// GetUserSessions retrieves all active sessions for a user
func (r *SessionRepository) GetUserSessions(userID uuid.UUID) ([]*models.Session, error) {
	query := `
		SELECT id, user_id, token_hash, refresh_token_hash,
			   device_info, user_agent, ip_address,
			   expires_at, refresh_expires_at, last_used_at,
			   is_active, revoked_at, revoked_reason,
			   created_at, updated_at
		FROM user_sessions 
		WHERE user_id = $1 AND is_active = true
		ORDER BY last_used_at DESC`

	rows, err := r.db.Query(query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user sessions: %w", err)
	}
	defer rows.Close()

	var sessions []*models.Session
	for rows.Next() {
		session := &models.Session{}
		err := rows.Scan(
			&session.ID,
			&session.UserID,
			&session.TokenHash,
			&session.RefreshTokenHash,
			&session.DeviceInfo,
			&session.UserAgent,
			&session.IPAddress,
			&session.ExpiresAt,
			&session.RefreshExpiresAt,
			&session.LastUsedAt,
			&session.IsActive,
			&session.RevokedAt,
			&session.RevokedReason,
			&session.CreatedAt,
			&session.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan session: %w", err)
		}
		sessions = append(sessions, session)
	}

	return sessions, nil
}

// CleanupExpiredSessions removes expired sessions
func (r *SessionRepository) CleanupExpiredSessions() error {
	query := `
		DELETE FROM user_sessions 
		WHERE expires_at < NOW() - INTERVAL '7 days'
		AND (refresh_expires_at IS NULL OR refresh_expires_at < NOW() - INTERVAL '7 days')`

	_, err := r.db.Exec(query)
	if err != nil {
		return fmt.Errorf("failed to cleanup expired sessions: %w", err)
	}

	return nil
}

// Redis cache operations
func (r *SessionRepository) cacheSession(session *models.Session) {
	ctx := context.Background()
	key := fmt.Sprintf("session:%s", session.TokenHash)
	
	// Cache for the duration until expiry
	duration := time.Until(session.ExpiresAt)
	if duration > 0 {
		r.redis.Set(ctx, key, session.ID.String(), duration)
	}
}

func (r *SessionRepository) getSessionFromCache(tokenHash string) (*models.Session, error) {
	ctx := context.Background()
	key := fmt.Sprintf("session:%s", tokenHash)
	
	sessionID, err := r.redis.Get(ctx, key).Result()
	if err != nil {
		return nil, err
	}

	// Get full session from database using ID
	id, err := uuid.Parse(sessionID)
	if err != nil {
		return nil, err
	}

	return r.GetByID(id)
}

func (r *SessionRepository) GetByID(id uuid.UUID) (*models.Session, error) {
	session := &models.Session{}
	query := `
		SELECT id, user_id, token_hash, refresh_token_hash,
			   device_info, user_agent, ip_address,
			   expires_at, refresh_expires_at, last_used_at,
			   is_active, revoked_at, revoked_reason,
			   created_at, updated_at
		FROM user_sessions 
		WHERE id = $1`

	err := r.db.QueryRow(query, id).Scan(
		&session.ID,
		&session.UserID,
		&session.TokenHash,
		&session.RefreshTokenHash,
		&session.DeviceInfo,
		&session.UserAgent,
		&session.IPAddress,
		&session.ExpiresAt,
		&session.RefreshExpiresAt,
		&session.LastUsedAt,
		&session.IsActive,
		&session.RevokedAt,
		&session.RevokedReason,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("session not found")
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	return session, nil
}

func (r *SessionRepository) removeSessionFromCache(sessionID string) {
	ctx := context.Background()
	pattern := fmt.Sprintf("session:*")
	
	// This is a simplified approach - in production, you'd want a more efficient way
	// to map session IDs to token hashes for cache invalidation
	keys, err := r.redis.Keys(ctx, pattern).Result()
	if err != nil {
		return
	}

	for _, key := range keys {
		cachedID, err := r.redis.Get(ctx, key).Result()
		if err == nil && cachedID == sessionID {
			r.redis.Del(ctx, key)
		}
	}
}

func (r *SessionRepository) clearUserSessionsFromCache(userID uuid.UUID) {
	// In a production system, you'd maintain a mapping of user IDs to session tokens
	// For now, this is a placeholder
	ctx := context.Background()
	pattern := fmt.Sprintf("session:*")
	r.redis.Del(ctx, pattern)
}


