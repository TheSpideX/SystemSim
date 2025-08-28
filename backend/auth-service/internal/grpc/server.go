package grpc

import (
	"fmt"
	"log"
	"net"
	"time"

	auth "github.com/systemsim/auth-service/internal/proto"
	"github.com/systemsim/auth-service/internal/config"
	"github.com/systemsim/auth-service/internal/grpc/handlers"
	"github.com/systemsim/auth-service/internal/health"
	"github.com/systemsim/auth-service/internal/services"
	"google.golang.org/grpc"
	grpchealth "google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// Server represents the gRPC server
type Server struct {
	server       *grpc.Server
	listener     net.Listener
	authHandler  *handlers.AuthGRPCHandler
	config       *config.Config
	healthServer *grpchealth.Server
}

// NewServer creates a new gRPC server
func NewServer(
	cfg *config.Config,
	authService *services.AuthService,
	rbacService *services.RBACService,
	userService *services.UserService,
	enhancedHealthChecker *health.EnhancedHealthChecker,
) (*Server, error) {
	// Create listener
	listener, err := net.Listen("tcp", ":"+cfg.GRPC.Port)
	if err != nil {
		return nil, fmt.Errorf("failed to listen on port %s: %w", cfg.GRPC.Port, err)
	}

	// Configure gRPC server options
	opts := []grpc.ServerOption{
		grpc.MaxRecvMsgSize(cfg.GRPC.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(cfg.GRPC.MaxSendMsgSize),
		grpc.ConnectionTimeout(cfg.GRPC.ConnectionTimeout),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			Time:    cfg.GRPC.KeepaliveTime,
			Timeout: cfg.GRPC.KeepaliveTimeout,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             10 * time.Second,
			PermitWithoutStream: true,
		}),
	}

	// Create gRPC server
	server := grpc.NewServer(opts...)

	// Create auth handler
	authHandler := handlers.NewAuthGRPCHandler(authService, rbacService, userService, enhancedHealthChecker)

	// Register services
	auth.RegisterAuthServiceServer(server, authHandler)

	// Register standard gRPC health service
	healthServer := grpchealth.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthServer)

	// Set initial health status for the auth service
	healthServer.SetServingStatus("auth", grpc_health_v1.HealthCheckResponse_SERVING) // Match the service name used by health checker
	healthServer.SetServingStatus("", grpc_health_v1.HealthCheckResponse_SERVING)     // Overall service health

	// Enable reflection for development
	if cfg.Server.Mode == "development" {
		reflection.Register(server)
	}

	return &Server{
		server:       server,
		listener:     listener,
		authHandler:  authHandler,
		config:       cfg,
		healthServer: healthServer,
	}, nil
}

// Start starts the gRPC server
func (s *Server) Start() error {
	log.Printf("Starting gRPC server on port %s", s.config.GRPC.Port)
	
	if err := s.server.Serve(s.listener); err != nil {
		return fmt.Errorf("failed to serve gRPC: %w", err)
	}
	
	return nil
}

// Stop gracefully stops the gRPC server
func (s *Server) Stop() {
	log.Println("Stopping gRPC server...")
	s.server.GracefulStop()
}

// GetPort returns the port the server is listening on
func (s *Server) GetPort() string {
	return s.config.GRPC.Port
}

// GetListener returns the server listener
func (s *Server) GetListener() net.Listener {
	return s.listener
}
