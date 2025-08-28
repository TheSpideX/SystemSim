package testfactory

import (
	"database/sql"
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/systemsim/auth-service/internal/config"
	"github.com/systemsim/auth-service/internal/events"
	"github.com/systemsim/auth-service/internal/repository"
	"github.com/systemsim/auth-service/internal/services"
	"github.com/systemsim/auth-service/internal/testutils"
)

// TestServices holds all service instances for testing
type TestServices struct {
	AuthService     *services.AuthService
	UserService     *services.UserService
	RBACService     *services.RBACService
	EventPublisher  *events.Publisher
	EventSubscriber *events.Subscriber
}

// CreateTestServices creates all service instances for testing
func CreateTestServices(t *testing.T, db *sql.DB, redisClient *redis.Client) (*TestServices, *testutils.TestRepositories) {
	cfg := testutils.SetupTestConfig()

	// Create repositories
	repos := testutils.SetupTestRepositories(t, db, redisClient)

	// Create event system
	eventPublisher := events.NewPublisher(redisClient)
	eventSubscriber := events.NewSubscriber(redisClient)

	// Create services
	rbacService := services.NewRBACService(repos.RBACRepo, repos.UserRepo)
	authService := services.NewAuthService(repos.UserRepo, repos.SessionRepo, rbacService, cfg.JWT, eventPublisher)
	userService := services.NewUserService(repos.UserRepo, repos.SessionRepo)

	return &TestServices{
		AuthService:     authService,
		UserService:     userService,
		RBACService:     rbacService,
		EventPublisher:  eventPublisher,
		EventSubscriber: eventSubscriber,
	}, repos
}

// CreateAuthService creates just the auth service for focused testing
func CreateAuthService(t *testing.T, db *sql.DB, redisClient *redis.Client) (*services.AuthService, *testutils.TestRepositories) {
	cfg := testutils.SetupTestConfig()
	repos := testutils.SetupTestRepositories(t, db, redisClient)
	
	eventPublisher := events.NewPublisher(redisClient)
	rbacService := services.NewRBACService(repos.RBACRepo, repos.UserRepo)
	authService := services.NewAuthService(repos.UserRepo, repos.SessionRepo, rbacService, cfg.JWT, eventPublisher)
	
	return authService, repos
}

// CreateRBACService creates just the RBAC service for focused testing
func CreateRBACService(t *testing.T, db *sql.DB, redisClient *redis.Client) (*services.RBACService, *testutils.TestRepositories) {
	repos := testutils.SetupTestRepositories(t, db, redisClient)
	rbacService := services.NewRBACService(repos.RBACRepo, repos.UserRepo)
	
	return rbacService, repos
}

// CreateUserService creates just the user service for focused testing
func CreateUserService(t *testing.T, db *sql.DB, redisClient *redis.Client) (*services.UserService, *testutils.TestRepositories) {
	repos := testutils.SetupTestRepositories(t, db, redisClient)
	userService := services.NewUserService(repos.UserRepo, repos.SessionRepo)
	
	return userService, repos
}
