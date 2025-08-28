package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
)

// Config holds all configuration for the API Gateway
type Config struct {
	Server   ServerConfig
	Services ServicesConfig
	Redis    RedisConfig
	Security SecurityConfig
}

// ServerConfig holds HTTP/2 server configuration
type ServerConfig struct {
	Port         string
	Host         string
	Mode         string // "development", "production"
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	// HTTP/2 configuration (strict HTTP/2-only)
	HTTP2Enabled         bool
	MaxConcurrentStreams uint32
	MaxFrameSize         uint32
	HTTP2IdleTimeout     time.Duration

	// TLS configuration (mandatory for HTTP/2)
	TLSEnabled    bool
	CertFile      string
	KeyFile       string
	MinTLSVersion string // "1.2" or "1.3"

	// Performance tuning
	MaxRequestBodySize int64
	CompressResponse   bool
	KeepAlive          bool
}

// ServicesConfig holds backend service connection info
type ServicesConfig struct {
	AuthService       ServiceConfig
	ProjectService    ServiceConfig
	SimulationService ServiceConfig
}

// ServiceConfig holds individual service configuration
type ServiceConfig struct {
	GRPCAddress       string
	ConnectionTimeout time.Duration
	RequestTimeout    time.Duration
	MaxConnections    int
	KeepAlive         bool
}

// RedisConfig holds Redis configuration for real-time events
type RedisConfig struct {
	Address     string
	Password    string
	DB          int
	PoolSize    int
	MaxRetries  int
	DialTimeout time.Duration
}

// SecurityConfig holds security-related configuration
type SecurityConfig struct {
	RateLimitEnabled bool
	RateLimitRPS     int
	CORSEnabled      bool
	AllowedOrigins   []string
	JWTSecret        string
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	// Load .env file if it exists (development)
	_ = godotenv.Load()

	cfg := &Config{
		Server: ServerConfig{
			Port:         getEnv("SERVER_PORT", "8000"),
			Host:         getEnv("SERVER_HOST", "0.0.0.0"),
			Mode:         getEnv("SERVER_MODE", "development"),
			ReadTimeout:  getDurationEnv("SERVER_READ_TIMEOUT", 10*time.Second),
			WriteTimeout: getDurationEnv("SERVER_WRITE_TIMEOUT", 10*time.Second),
			IdleTimeout:  getDurationEnv("SERVER_IDLE_TIMEOUT", 60*time.Second),

			// Strict HTTP/2 configuration
			HTTP2Enabled:         getBoolEnv("HTTP2_ENABLED", true),
			MaxConcurrentStreams: uint32(getIntEnv("HTTP2_MAX_CONCURRENT_STREAMS", 1000)),
			MaxFrameSize:         uint32(getIntEnv("HTTP2_MAX_FRAME_SIZE", 16384)),
			HTTP2IdleTimeout:     getDurationEnv("HTTP2_IDLE_TIMEOUT", 60*time.Second),

			// Mandatory TLS for HTTP/2
			TLSEnabled:    getBoolEnv("TLS_ENABLED", true),
			CertFile:      getEnv("TLS_CERT_FILE", "certs/server.crt"),
			KeyFile:       getEnv("TLS_KEY_FILE", "certs/server.key"),
			MinTLSVersion: getEnv("TLS_MIN_VERSION", "1.2"),

			// Performance optimization
			MaxRequestBodySize: int64(getIntEnv("MAX_REQUEST_BODY_SIZE", 10*1024*1024)), // 10MB
			CompressResponse:   getBoolEnv("COMPRESS_RESPONSE", true),
			KeepAlive:          getBoolEnv("KEEP_ALIVE", true),
		},
		Services: ServicesConfig{
			AuthService: ServiceConfig{
				GRPCAddress:       getEnv("AUTH_SERVICE_GRPC", "localhost:9000"),
				ConnectionTimeout: getDurationEnv("AUTH_CONNECTION_TIMEOUT", 5*time.Second),
				RequestTimeout:    getDurationEnv("AUTH_REQUEST_TIMEOUT", 10*time.Second),
				MaxConnections:    getIntEnv("AUTH_MAX_CONNECTIONS", 20),
				KeepAlive:         getBoolEnv("AUTH_KEEP_ALIVE", true),
			},
			ProjectService: ServiceConfig{
				GRPCAddress:       getEnv("PROJECT_SERVICE_GRPC", "localhost:10000"),
				ConnectionTimeout: getDurationEnv("PROJECT_CONNECTION_TIMEOUT", 5*time.Second),
				RequestTimeout:    getDurationEnv("PROJECT_REQUEST_TIMEOUT", 15*time.Second),
				MaxConnections:    getIntEnv("PROJECT_MAX_CONNECTIONS", 15),
				KeepAlive:         getBoolEnv("PROJECT_KEEP_ALIVE", true),
			},
			SimulationService: ServiceConfig{
				GRPCAddress:       getEnv("SIMULATION_SERVICE_GRPC", "localhost:11000"),
				ConnectionTimeout: getDurationEnv("SIMULATION_CONNECTION_TIMEOUT", 5*time.Second),
				RequestTimeout:    getDurationEnv("SIMULATION_REQUEST_TIMEOUT", 30*time.Second),
				MaxConnections:    getIntEnv("SIMULATION_MAX_CONNECTIONS", 25),
				KeepAlive:         getBoolEnv("SIMULATION_KEEP_ALIVE", true),
			},
		},
		Redis: RedisConfig{
			Address:     getEnv("REDIS_ADDRESS", "localhost:6379"),
			Password:    getEnv("REDIS_PASSWORD", ""),
			DB:          getIntEnv("REDIS_DB", 0),
			PoolSize:    getIntEnv("REDIS_POOL_SIZE", 100),
			MaxRetries:  getIntEnv("REDIS_MAX_RETRIES", 3),
			DialTimeout: getDurationEnv("REDIS_DIAL_TIMEOUT", 5*time.Second),
		},
		Security: SecurityConfig{
			RateLimitEnabled: getBoolEnv("RATE_LIMIT_ENABLED", true),
			RateLimitRPS:     getIntEnv("RATE_LIMIT_RPS", 1000),
			CORSEnabled:      getBoolEnv("CORS_ENABLED", true),
			AllowedOrigins:   getSliceEnv("ALLOWED_ORIGINS", []string{"https://localhost:3000"}),
			JWTSecret:        getEnv("JWT_SECRET", "your-secret-key"),
		},
	}

	return cfg, nil
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

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
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

func getSliceEnv(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		// Simple comma-separated parsing
		return []string{value} // Simplified for now
	}
	return defaultValue
}
