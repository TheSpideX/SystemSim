package testutils

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
	"github.com/systemsim/auth-service/internal/config"
	"github.com/systemsim/auth-service/internal/database"
	"github.com/systemsim/auth-service/internal/events"
	"github.com/systemsim/auth-service/internal/models"
	"github.com/systemsim/auth-service/internal/repository"
	"golang.org/x/crypto/bcrypt"
	_ "github.com/lib/pq"
)

// TestConfig holds test configuration
type TestConfig struct {
	DB    *sql.DB
	Redis *redis.Client
	Config *config.Config
}

// TestRepositories holds all repository instances for testing
type TestRepositories struct {
	UserRepo    *repository.UserRepository
	SessionRepo *repository.SessionRepository
	RBACRepo    *repository.RBACRepository
}

// SetupTestDB creates a test database connection
func SetupTestDB(t *testing.T) *sql.DB {
	// Use environment variables or defaults for test database
	dbHost := getEnv("TEST_DB_HOST", "localhost")
	dbPort := getEnv("TEST_DB_PORT", "5432")
	dbUser := getEnv("TEST_DB_USER", "postgres")
	dbPassword := getEnv("TEST_DB_PASSWORD", "password")
	dbName := getEnv("TEST_DB_NAME", "auth_service_test")

	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err, "Failed to connect to test database")

	// Test connection
	err = db.Ping()
	require.NoError(t, err, "Failed to ping test database")

	// Create schema
	err = database.EnsureSchema(db)
	require.NoError(t, err, "Failed to create test schema")

	return db
}

// SetupTestRedis creates a test Redis connection
func SetupTestRedis(t *testing.T) *redis.Client {
	redisHost := getEnv("TEST_REDIS_HOST", "localhost")
	redisPort := getEnv("TEST_REDIS_PORT", "6379")
	redisDB := 1 // Use DB 1 for tests

	client := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", redisHost, redisPort),
		DB:   redisDB,
	})

	// Test connection
	ctx := context.Background()
	_, err := client.Ping(ctx).Result()
	require.NoError(t, err, "Failed to connect to test Redis")

	return client
}

// SetupTestConfig creates a test configuration
func SetupTestConfig() *config.Config {
	return &config.Config{
		Database: config.DatabaseConfig{
			URL:             "postgres://postgres:password@localhost:5432/auth_service_test?sslmode=disable",
			MaxOpenConns:    10,
			MaxIdleConns:    5,
			ConnMaxLifetime: 5 * time.Minute,
			ConnMaxIdleTime: 5 * time.Minute,
		},
		Redis: config.RedisConfig{
			Addr:         "localhost:6379",
			Password:     "",
			DB:           1,
			PoolSize:     10,
			MinIdleConns: 5,
			MaxRetries:   3,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
			IdleTimeout:  5 * time.Minute,
		},
		JWT: config.JWTConfig{
			Secret:               "test-secret-key-for-testing-only",
			AccessTokenDuration:  15 * time.Minute,
			RefreshTokenDuration: 7 * 24 * time.Hour,
			Issuer:               "auth-service-test",
		},
		Server: config.ServerConfig{
			Port:         "9001",
			Mode:         "test",
			ReadTimeout:  10 * time.Second,
			WriteTimeout: 10 * time.Second,
			IdleTimeout:  60 * time.Second,
		},
		GRPC: config.GRPCConfig{
			Port: "9000",
		},
	}
}

// SetupTestRepositories creates all repository instances for testing
func SetupTestRepositories(t *testing.T, db *sql.DB, redisClient *redis.Client) *TestRepositories {
	// Create repositories
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db, redisClient)
	rbacRepo := repository.NewRBACRepository(db)

	return &TestRepositories{
		UserRepo:    userRepo,
		SessionRepo: sessionRepo,
		RBACRepo:    rbacRepo,
	}
}

// CleanupTestDB cleans up test database
func CleanupTestDB(t *testing.T, db *sql.DB) {
	// Clean up all tables in reverse order of dependencies
	tables := []string{
		"user_roles",
		"user_sessions", 
		"permissions",
		"roles",
		"users",
	}

	for _, table := range tables {
		_, err := db.Exec(fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			log.Printf("Warning: Failed to clean table %s: %v", table, err)
		}
	}
}

// CleanupTestRedis cleans up test Redis
func CleanupTestRedis(t *testing.T, client *redis.Client) {
	ctx := context.Background()
	err := client.FlushDB(ctx).Err()
	if err != nil {
		log.Printf("Warning: Failed to flush test Redis: %v", err)
	}
}

// CreateTestUser creates a test user in the database
func CreateTestUser(t *testing.T, userRepo *repository.UserRepository, email string) *models.User {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("testpassword123"), bcrypt.DefaultCost)
	require.NoError(t, err)

	user := &models.User{
		ID:            uuid.New(),
		Email:         email,
		PasswordHash:  string(hashedPassword),
		FirstName:     "Test",
		LastName:      "User",
		Company:       "Test Company",
		EmailVerified: true,
		IsActive:      true,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	err = userRepo.Create(user)
	require.NoError(t, err)

	return user
}

// CreateTestRole creates a test role in the database
func CreateTestRole(t *testing.T, db *sql.DB, name, description string) *models.Role {
	role := &models.Role{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		IsSystem:    false,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// Insert role directly using SQL
	query := `
		INSERT INTO roles (id, name, description, is_system, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
	`
	_, err := db.Exec(query,
		role.ID, role.Name, role.Description, role.IsSystem, role.CreatedAt, role.UpdatedAt)
	require.NoError(t, err)

	return role
}

// CreateTestPermission creates a test permission in the database
func CreateTestPermission(t *testing.T, db *sql.DB, name, description string) *models.Permission {
	permission := &models.Permission{
		ID:          uuid.New(),
		Name:        name,
		Description: description,
		CreatedAt:   time.Now(),
	}

	// Insert permission directly using SQL
	query := `
		INSERT INTO permissions (id, name, description, created_at)
		VALUES ($1, $2, $3, $4)
	`
	_, err := db.Exec(query,
		permission.ID, permission.Name, permission.Description, permission.CreatedAt)
	require.NoError(t, err)

	return permission
}

// AssertUserEqual compares two users for equality
func AssertUserEqual(t *testing.T, expected, actual *models.User) {
	require.Equal(t, expected.ID, actual.ID)
	require.Equal(t, expected.Email, actual.Email)
	require.Equal(t, expected.FirstName, actual.FirstName)
	require.Equal(t, expected.LastName, actual.LastName)
	require.Equal(t, expected.Company, actual.Company)
	require.Equal(t, expected.EmailVerified, actual.EmailVerified)
	require.Equal(t, expected.IsActive, actual.IsActive)
}

// getEnv gets environment variable with fallback
func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

// TestEmailProcessor implements EmailProcessor for testing
type TestEmailProcessor struct {
	SentEmails []events.EmailTask
	mutex      sync.Mutex
}

// ProcessWelcomeEmail implements the EmailProcessor interface
func (p *TestEmailProcessor) ProcessWelcomeEmail(task *events.EmailTask) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.SentEmails = append(p.SentEmails, *task)
	return nil
}

// ProcessVerificationEmail implements the EmailProcessor interface
func (p *TestEmailProcessor) ProcessVerificationEmail(task *events.EmailTask) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.SentEmails = append(p.SentEmails, *task)
	return nil
}

// ProcessPasswordResetEmail implements the EmailProcessor interface
func (p *TestEmailProcessor) ProcessPasswordResetEmail(task *events.EmailTask) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.SentEmails = append(p.SentEmails, *task)
	return nil
}

// ProcessNotificationEmail implements the EmailProcessor interface
func (p *TestEmailProcessor) ProcessNotificationEmail(task *events.EmailTask) error {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.SentEmails = append(p.SentEmails, *task)
	return nil
}

// GetLastEmail returns the last sent email (thread-safe)
func (p *TestEmailProcessor) GetLastEmail() *events.EmailTask {
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
