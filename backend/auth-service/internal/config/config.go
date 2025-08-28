package config

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the auth service
type Config struct {
	Server    ServerConfig
	GRPC      GRPCConfig
	Database  DatabaseConfig
	Redis     RedisConfig
	JWT       JWTConfig
	RateLimit RateLimitConfig
	Email     EmailConfig
	Mesh      MeshConfig
}

// ServerConfig holds HTTP/2 server configuration
type ServerConfig struct {
	Port         string
	Mode         string // "development", "production"
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	// HTTP/2 configuration
	HTTP2Enabled         bool
	MaxConcurrentStreams uint32
	MaxFrameSize         uint32
	HTTP2IdleTimeout     time.Duration

	// TLS configuration for HTTP/2
	TLSEnabled   bool
	CertFile     string
	KeyFile      string
	MinTLSVersion string // "1.2" or "1.3"
}

// GRPCConfig holds gRPC server configuration
type GRPCConfig struct {
	Port            string
	MaxRecvMsgSize  int
	MaxSendMsgSize  int
	ConnectionTimeout time.Duration
	KeepaliveTime   time.Duration
	KeepaliveTimeout time.Duration
}

// MeshConfig holds service mesh configuration
type MeshConfig struct {
	ServiceName     string
	InstanceID      string
	DiscoveryPrefix string
	HealthInterval  time.Duration
	ConnectionPools ConnectionPoolConfig
}

// ConnectionPoolConfig holds connection pool settings
type ConnectionPoolConfig struct {
	MinConnections    int
	MaxConnections    int
	MaxIdleTime       time.Duration
	HealthCheckInterval time.Duration
}

// DatabaseConfig holds database configuration
type DatabaseConfig struct {
	URL             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
}

// RedisConfig holds Redis configuration
type RedisConfig struct {
	Addr         string
	Password     string
	DB           int
	PoolSize     int
	MinIdleConns int
	MaxRetries   int
	DialTimeout  time.Duration
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
}

// JWTConfig holds JWT configuration
type JWTConfig struct {
	Secret               string
	AccessTokenDuration  time.Duration
	RefreshTokenDuration time.Duration
	Issuer               string
}

// RateLimitConfig holds rate limiting configuration
type RateLimitConfig struct {
	RequestsPerMinute int
	RedisAddr         string
	RedisPassword     string
	RedisDB           int
}

// EmailConfig holds email configuration
type EmailConfig struct {
	SMTPHost     string
	SMTPPort     int
	SMTPUsername string
	SMTPPassword string
	FromEmail    string
	FromName     string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("HTTP_PORT", "9001"),
			Mode:         getEnv("GIN_MODE", "development"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),

			// HTTP/2 configuration
			HTTP2Enabled:         getBoolEnv("HTTP2_ENABLED", true),
			MaxConcurrentStreams: uint32(getIntEnv("HTTP2_MAX_CONCURRENT_STREAMS", 250)),
			MaxFrameSize:         uint32(getIntEnv("HTTP2_MAX_FRAME_SIZE", 16384)),
			HTTP2IdleTimeout:     getDurationEnv("HTTP2_IDLE_TIMEOUT", 60*time.Second),

			// TLS configuration for HTTP/2
			TLSEnabled:    getBoolEnv("TLS_ENABLED", true), // Enable TLS by default for HTTP/2
			CertFile:      getEnv("TLS_CERT_FILE", "certs/server.crt"),
			KeyFile:       getEnv("TLS_KEY_FILE", "certs/server.key"),
			MinTLSVersion: getEnv("TLS_MIN_VERSION", "1.2"),
		},
		GRPC: GRPCConfig{
			Port:              getEnv("GRPC_PORT", "9000"),
			MaxRecvMsgSize:    getIntEnv("GRPC_MAX_RECV_MSG_SIZE", 4*1024*1024), // 4MB
			MaxSendMsgSize:    getIntEnv("GRPC_MAX_SEND_MSG_SIZE", 4*1024*1024), // 4MB
			ConnectionTimeout: getDurationEnv("GRPC_CONNECTION_TIMEOUT", 5*time.Second),
			KeepaliveTime:     getDurationEnv("GRPC_KEEPALIVE_TIME", 30*time.Second),
			KeepaliveTimeout:  getDurationEnv("GRPC_KEEPALIVE_TIMEOUT", 5*time.Second),
		},
		Database: DatabaseConfig{
			URL:             getEnv("DATABASE_URL", "postgres://auth_user:auth_password@localhost:5432/systemsim_auth?sslmode=disable"),
			MaxOpenConns:    getIntEnv("DB_MAX_OPEN_CONNS", 25),
			MaxIdleConns:    getIntEnv("DB_MAX_IDLE_CONNS", 5),
			ConnMaxLifetime: getDurationEnv("DB_CONN_MAX_LIFETIME", 5*time.Minute),
			ConnMaxIdleTime: getDurationEnv("DB_CONN_MAX_IDLE_TIME", 5*time.Minute),
		},
		Redis: RedisConfig{
			Addr:         getEnv("REDIS_ADDR", "localhost:6379"),
			Password:     getEnv("REDIS_PASSWORD", ""),
			DB:           getIntEnv("REDIS_DB", 0),
			PoolSize:     getIntEnv("REDIS_POOL_SIZE", 10),
			MinIdleConns: getIntEnv("REDIS_MIN_IDLE_CONNS", 2),
			MaxRetries:   getIntEnv("REDIS_MAX_RETRIES", 3),
			DialTimeout:  getDurationEnv("REDIS_DIAL_TIMEOUT", 5*time.Second),
			ReadTimeout:  getDurationEnv("REDIS_READ_TIMEOUT", 3*time.Second),
			WriteTimeout: getDurationEnv("REDIS_WRITE_TIMEOUT", 3*time.Second),
			IdleTimeout:  getDurationEnv("REDIS_IDLE_TIMEOUT", 5*time.Minute),
		},
		JWT: JWTConfig{
			Secret:               getEnv("JWT_SECRET", "your-super-secret-jwt-key-change-this-in-production"),
			AccessTokenDuration:  getDurationEnv("JWT_ACCESS_DURATION", 15*time.Minute),
			RefreshTokenDuration: getDurationEnv("JWT_REFRESH_DURATION", 7*24*time.Hour),
			Issuer:               getEnv("JWT_ISSUER", "systemsim-auth"),
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: getIntEnv("RATE_LIMIT_RPM", 60),
			RedisAddr:         getEnv("REDIS_ADDR", "localhost:6379"),
			RedisPassword:     getEnv("REDIS_PASSWORD", ""),
			RedisDB:           getIntEnv("REDIS_DB", 0),
		},
		Email: EmailConfig{
			SMTPHost:     getEnv("SMTP_HOST", "localhost"),
			SMTPPort:     getIntEnv("SMTP_PORT", 587),
			SMTPUsername: getEnv("SMTP_USERNAME", ""),
			SMTPPassword: getEnv("SMTP_PASSWORD", ""),
			FromEmail:    getEnv("FROM_EMAIL", "noreply@systemsim.app"),
			FromName:     getEnv("FROM_NAME", "SystemSim"),
		},
		Mesh: MeshConfig{
			ServiceName:     getEnv("MESH_SERVICE_NAME", "auth-service"),
			InstanceID:      getEnv("MESH_INSTANCE_ID", "auth-service-1"),
			DiscoveryPrefix: getEnv("MESH_DISCOVERY_PREFIX", "services"),
			HealthInterval:  getDurationEnv("MESH_HEALTH_INTERVAL", 30*time.Second),
			ConnectionPools: ConnectionPoolConfig{
				MinConnections:      getIntEnv("MESH_MIN_CONNECTIONS", 5),
				MaxConnections:      getIntEnv("MESH_MAX_CONNECTIONS", 20),
				MaxIdleTime:         getDurationEnv("MESH_MAX_IDLE_TIME", 5*time.Minute),
				HealthCheckInterval: getDurationEnv("MESH_HEALTH_CHECK_INTERVAL", 30*time.Second),
			},
		},
	}

	// Validate required configuration
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return cfg, nil
}

// validate validates the configuration
func (c *Config) validate() error {
	if c.JWT.Secret == "your-super-secret-jwt-key-change-this-in-production" {
		return fmt.Errorf("JWT secret must be changed from default value")
	}

	if len(c.JWT.Secret) < 32 {
		return fmt.Errorf("JWT secret must be at least 32 characters long")
	}

	if c.Database.URL == "" {
		return fmt.Errorf("database URL is required")
	}

	// HTTP/2 only validation - no fallback allowed
	if !c.Server.HTTP2Enabled {
		return fmt.Errorf("HTTP/2 must be enabled - HTTP/1.1 fallback not supported")
	}

	if !c.Server.TLSEnabled {
		return fmt.Errorf("TLS must be enabled for HTTP/2-only operation")
	}

	return nil
}

// Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}
