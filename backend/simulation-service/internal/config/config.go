package config

import (
	"fmt"
	"os"
	"strconv"
	"time"
)

// Config holds all configuration for the simulation service
type Config struct {
	Server     ServerConfig     `json:"server"`
	GRPC       GRPCConfig       `json:"grpc"`
	Redis      RedisConfig      `json:"redis"`
	Simulation SimulationConfig `json:"simulation"`
	Mesh       MeshConfig       `json:"mesh"`
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	Port        int    `json:"port"`
	Environment string `json:"environment"`
	Host        string `json:"host"`
}

// GRPCConfig holds gRPC server configuration
type GRPCConfig struct {
	Port int    `json:"port"`
	Host string `json:"host"`
}

// RedisConfig holds Redis connection configuration
type RedisConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Password string `json:"password"`
	DB       int    `json:"db"`
}

// SimulationConfig holds simulation-specific configuration
type SimulationConfig struct {
	TickDuration        time.Duration `json:"tick_duration"`
	MaxSimulations      int           `json:"max_simulations"`
	MaxComponentsPerSim int           `json:"max_components_per_sim"`
	DefaultTimeout      time.Duration `json:"default_timeout"`
	ProfilesPath        string        `json:"profiles_path"`
	TemplatesPath       string        `json:"templates_path"`
}

// MeshConfig holds microservice mesh configuration
type MeshConfig struct {
	ServiceName     string            `json:"service_name"`
	ServiceVersion  string            `json:"service_version"`
	DiscoveryPort   int               `json:"discovery_port"`
	HealthCheckPort int               `json:"health_check_port"`
	Services        map[string]string `json:"services"`
}

// Load loads configuration from environment variables with defaults
func Load() (*Config, error) {
	config := &Config{
		Server: ServerConfig{
			Port:        getEnvAsInt("SERVER_PORT", 11000),
			Environment: getEnv("ENVIRONMENT", "development"),
			Host:        getEnv("SERVER_HOST", "0.0.0.0"),
		},
		GRPC: GRPCConfig{
			Port: getEnvAsInt("GRPC_PORT", 11001),
			Host: getEnv("GRPC_HOST", "0.0.0.0"),
		},
		Redis: RedisConfig{
			Host:     getEnv("REDIS_HOST", "localhost"),
			Port:     getEnvAsInt("REDIS_PORT", 6379),
			Password: getEnv("REDIS_PASSWORD", ""),
			DB:       getEnvAsInt("REDIS_DB", 0),
		},
		Simulation: SimulationConfig{
			TickDuration:        getEnvAsDuration("TICK_DURATION", 10*time.Microsecond), // 0.01ms
			MaxSimulations:      getEnvAsInt("MAX_SIMULATIONS", 100),
			MaxComponentsPerSim: getEnvAsInt("MAX_COMPONENTS_PER_SIM", 10000),
			DefaultTimeout:      getEnvAsDuration("DEFAULT_TIMEOUT", 30*time.Second),
			ProfilesPath:        getEnv("PROFILES_PATH", "./profiles"),
			TemplatesPath:       getEnv("TEMPLATES_PATH", "./templates"),
		},
		Mesh: MeshConfig{
			ServiceName:     getEnv("SERVICE_NAME", "simulation-service"),
			ServiceVersion:  getEnv("SERVICE_VERSION", "1.0.0"),
			DiscoveryPort:   getEnvAsInt("DISCOVERY_PORT", 11002),
			HealthCheckPort: getEnvAsInt("HEALTH_CHECK_PORT", 11003),
			Services: map[string]string{
				"auth-service": getEnv("AUTH_SERVICE_URL", "localhost:8001"),
			},
		},
	}

	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	return config, nil
}

// validateConfig validates the loaded configuration
func validateConfig(config *Config) error {
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("invalid server port: %d", config.Server.Port)
	}

	if config.GRPC.Port <= 0 || config.GRPC.Port > 65535 {
		return fmt.Errorf("invalid gRPC port: %d", config.GRPC.Port)
	}

	if config.Redis.Port <= 0 || config.Redis.Port > 65535 {
		return fmt.Errorf("invalid Redis port: %d", config.Redis.Port)
	}

	if config.Simulation.TickDuration <= 0 {
		return fmt.Errorf("tick duration must be positive")
	}

	if config.Simulation.MaxSimulations <= 0 {
		return fmt.Errorf("max simulations must be positive")
	}

	if config.Simulation.MaxComponentsPerSim <= 0 {
		return fmt.Errorf("max components per simulation must be positive")
	}

	return nil
}

// Helper functions for environment variable parsing
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvAsDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// GetRedisAddr returns the Redis connection address
func (r *RedisConfig) GetRedisAddr() string {
	return fmt.Sprintf("%s:%d", r.Host, r.Port)
}

// GetServerAddr returns the HTTP server address
func (s *ServerConfig) GetServerAddr() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// GetGRPCAddr returns the gRPC server address
func (g *GRPCConfig) GetGRPCAddr() string {
	return fmt.Sprintf("%s:%d", g.Host, g.Port)
}
