package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"strconv"
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/systemsim/auth-service/internal/config"
	"github.com/systemsim/auth-service/internal/database"
	"github.com/systemsim/auth-service/internal/discovery"
	"github.com/systemsim/auth-service/internal/events"
	grpcserver "github.com/systemsim/auth-service/internal/grpc"
	"github.com/systemsim/auth-service/internal/handlers"
	"github.com/systemsim/auth-service/internal/health"
	"github.com/systemsim/auth-service/internal/http2"
	"github.com/systemsim/auth-service/internal/mesh"
	"github.com/systemsim/auth-service/internal/metrics"
	"github.com/systemsim/auth-service/internal/middleware"
	"github.com/systemsim/auth-service/internal/repository"
	"github.com/systemsim/auth-service/internal/services"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize database
	db, err := database.NewPostgresDB(cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Ensure database schema exists
	if err := database.EnsureSchema(db); err != nil {
		log.Fatalf("Failed to ensure database schema: %v", err)
	}

	// Initialize Redis
	redisClient, err := database.NewRedisClient(cfg.Redis)
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	defer redisClient.Close()

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db, redisClient)
	rbacRepo := repository.NewRBACRepository(db)

	// Initialize health monitoring
	healthChecker := health.NewHealthChecker(db, redisClient, "1.0.0")
	metrics.InitializeMetrics()

	// Initialize event system
	eventPublisher := events.NewPublisher(redisClient)
	eventSubscriber := events.NewSubscriber(redisClient)
	emailProcessor := &events.MockEmailProcessor{} // TODO: Replace with real email processor

	// Initialize services
	rbacService := services.NewRBACService(rbacRepo, userRepo)
	authService := services.NewAuthService(userRepo, sessionRepo, rbacService, cfg.JWT, eventPublisher)
	userService := services.NewUserService(userRepo, sessionRepo)

	// Initialize service registry and discovery
	grpcPort, _ := strconv.Atoi(cfg.GRPC.Port)
	http2Port, _ := strconv.Atoi(cfg.Server.Port)
	serviceInfo := discovery.CreateAuthServiceInfo(grpcPort, http2Port, "1.0.0")
	serviceRegistry := discovery.NewServiceRegistry(redisClient, serviceInfo)
	serviceDiscovery := discovery.NewServiceDiscovery(redisClient)

	// Initialize connection pool manager (5-20 connections per service)
	poolManager := mesh.DefaultPoolManager(serviceDiscovery)
	meshClient := mesh.NewMeshClientWithCircuitBreaker(poolManager)

	// Initialize enhanced health checker
	enhancedHealthChecker := health.NewEnhancedHealthChecker(db, redisClient, eventPublisher, eventSubscriber, meshClient, "1.0.0")

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	userHandler := handlers.NewUserHandler(userService, authService)
	rbacHandler := handlers.NewRBACHandler(rbacService)
	healthHandler := handlers.NewHealthHandler(healthChecker, enhancedHealthChecker)

	// Initialize middleware
	rbacMiddleware := middleware.NewRBACMiddleware(rbacService)

	// Setup HTTP router
	router := setupRouter(cfg, authHandler, userHandler, rbacHandler, healthHandler, rbacMiddleware)

	// Create HTTP/2 server
	http2Server, err := http2.NewServer(cfg, router)
	if err != nil {
		log.Fatalf("Failed to create HTTP/2 server: %v", err)
	}

	// Create gRPC server
	grpcServer, err := grpcserver.NewServer(cfg, authService, rbacService, userService, enhancedHealthChecker)
	if err != nil {
		log.Fatalf("Failed to create gRPC server: %v", err)
	}

	// Start both servers (always enabled)
	var wg sync.WaitGroup

	// Start HTTP/2 server (for API Gateway communication)
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := http2Server.Start(); err != nil {
			log.Printf("HTTP/2 server error: %v", err)
		}
	}()

	// Start gRPC server (for internal mesh communication)
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("gRPC server starting on port %s (internal mesh communication)", cfg.GRPC.Port)
		if err := grpcServer.Start(); err != nil {
			log.Printf("gRPC server error: %v", err)
		}
	}()

	// Start event subscriber (for background processing)
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Event subscriber starting (background processing)")
		if err := eventSubscriber.Start(emailProcessor); err != nil {
			log.Printf("Event subscriber error: %v", err)
		}
	}()

	// Start service registry (register in mesh network)
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Service registry starting (mesh registration)")
		if err := serviceRegistry.Start(); err != nil {
			log.Printf("Service registry error: %v", err)
		}
	}()

	// Start connection pool manager (dynamic gRPC connection pools)
	wg.Add(1)
	go func() {
		defer wg.Done()
		log.Printf("Connection pool manager starting (5-20 connections per service)")
		if err := poolManager.Start(); err != nil {
			log.Printf("Connection pool manager error: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down servers...")

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown HTTP/2 server
	if err := http2Server.Shutdown(ctx); err != nil {
		log.Printf("HTTP/2 server forced to shutdown: %v", err)
	}

	// Shutdown gRPC server
	grpcServer.Stop()

	// Shutdown event subscriber
	if err := eventSubscriber.Stop(); err != nil {
		log.Printf("Event subscriber forced to shutdown: %v", err)
	}

	// Shutdown service registry
	if err := serviceRegistry.Stop(); err != nil {
		log.Printf("Service registry forced to shutdown: %v", err)
	}

	// Shutdown connection pool manager
	if err := poolManager.Stop(); err != nil {
		log.Printf("Connection pool manager forced to shutdown: %v", err)
	}

	// Wait for both servers to finish
	wg.Wait()
	log.Println("Servers exited")
}

func setupRouter(cfg *config.Config, authHandler *handlers.AuthHandler, userHandler *handlers.UserHandler, rbacHandler *handlers.RBACHandler, healthHandler *handlers.HealthHandler, rbacMiddleware *middleware.RBACMiddleware) *gin.Engine {
	if cfg.Server.Mode == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()

	// Global middleware
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.SecurityHeaders())
	// router.Use(middleware.RateLimiter(cfg.RateLimit)) // DISABLED FOR LOAD TESTING
	router.Use(metrics.HTTPMetricsMiddleware())

	// Health check endpoints
	router.GET("/health", healthHandler.SimpleHealthCheck)
	router.GET("/health/live", healthHandler.LivenessCheck)
	router.GET("/health/ready", healthHandler.ReadinessCheck)
	router.GET("/health/detailed", healthHandler.DetailedHealthCheck)
	router.GET("/metrics", healthHandler.MetricsHandler)

	// API routes
	api := router.Group("/api/v1")
	{
		// Public routes
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
			auth.POST("/refresh", authHandler.RefreshToken)
			auth.POST("/forgot-password", authHandler.ForgotPassword)
			auth.POST("/reset-password", authHandler.ResetPassword)
			auth.POST("/verify-email", authHandler.VerifyEmail)
			auth.POST("/resend-verification", authHandler.ResendVerificationEmail)
		}

		// Protected routes
		protected := api.Group("/")
		protected.Use(middleware.AuthRequired(cfg.JWT.Secret))
		protected.Use(rbacMiddleware.AddUserPermissions())
		{
			protected.POST("/auth/logout", authHandler.Logout)
			protected.GET("/user/profile", userHandler.GetProfile)
			protected.PUT("/user/profile", userHandler.UpdateProfile)
			protected.POST("/user/change-password", userHandler.ChangePassword)
			protected.DELETE("/user/account", userHandler.DeleteAccount)

			// Session management
			protected.GET("/user/sessions", userHandler.GetSessions)
			protected.DELETE("/user/sessions/:sessionId", userHandler.RevokeSession)
			protected.DELETE("/user/sessions", userHandler.RevokeAllSessions)
			protected.GET("/user/stats", userHandler.GetStats)

			// RBAC endpoints
			protected.GET("/rbac/my-roles", rbacHandler.GetMyRoles)
			protected.GET("/rbac/my-permissions", rbacHandler.GetMyPermissions)

			// Admin RBAC endpoints
			admin := protected.Group("/admin")
			admin.Use(rbacMiddleware.RequireAdmin())
			{
				admin.GET("/roles", rbacHandler.GetAllRoles)
				admin.GET("/permissions", rbacHandler.GetAllPermissions)
				admin.POST("/users/assign-role", rbacHandler.AssignRole)
				admin.POST("/users/remove-role", rbacHandler.RemoveRole)
				admin.GET("/users/:userId/roles", rbacHandler.GetUserRoles)
			}
		}
	}

	return router
}
