package health

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/systemsim/auth-service/internal/events"
)

// EnhancedHealthChecker provides comprehensive health monitoring
type EnhancedHealthChecker struct {
	db              *sql.DB
	redis           *redis.Client
	eventPublisher  *events.Publisher
	eventSubscriber *events.Subscriber
	meshClient      MeshHealthChecker
	version         string
	startTime       time.Time
}

// MeshHealthChecker interface for mesh client health checking
type MeshHealthChecker interface {
	HealthCheck() map[string]interface{}
}

// NewEnhancedHealthChecker creates a new enhanced health checker
func NewEnhancedHealthChecker(db *sql.DB, redis *redis.Client, eventPublisher *events.Publisher, eventSubscriber *events.Subscriber, meshClient MeshHealthChecker, version string) *EnhancedHealthChecker {
	return &EnhancedHealthChecker{
		db:              db,
		redis:           redis,
		eventPublisher:  eventPublisher,
		eventSubscriber: eventSubscriber,
		meshClient:      meshClient,
		version:         version,
		startTime:       time.Now(),
	}
}

// EnhancedHealthStatus represents the overall enhanced health status
type EnhancedHealthStatus struct {
	Healthy        bool                   `json:"healthy"`
	Status         string                 `json:"status"`
	Version        string                 `json:"version"`
	Timestamp      int64                  `json:"timestamp"`
	Uptime         string                 `json:"uptime"`
	ResponseTimeMs float64                `json:"response_time_ms"`
	Details        *DetailedHealthStatus  `json:"details"`
}

// DetailedHealthStatus provides detailed component health information
type DetailedHealthStatus struct {
	Database     *ComponentHealth `json:"database"`
	Redis        *ComponentHealth `json:"redis"`
	EventSystem  *ComponentHealth `json:"event_system"`
	MeshNetwork  *ComponentHealth `json:"mesh_network"`
	System       *SystemHealth    `json:"system"`
	Dependencies *DependencyHealth `json:"dependencies"`
}

// ComponentHealth represents health status of a component
type ComponentHealth struct {
	Healthy      bool    `json:"healthy"`
	Status       string  `json:"status"`
	ResponseTime float64 `json:"response_time_ms"`
	Error        string  `json:"error,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
}

// SystemHealth represents system-level health metrics
type SystemHealth struct {
	MemoryUsageMB    float64 `json:"memory_usage_mb"`
	MemoryUsagePercent float64 `json:"memory_usage_percent"`
	GoroutineCount   int     `json:"goroutine_count"`
	CPUCount         int     `json:"cpu_count"`
}

// DependencyHealth represents health of external dependencies
type DependencyHealth struct {
	TotalDependencies int `json:"total_dependencies"`
	HealthyCount      int `json:"healthy_count"`
	UnhealthyCount    int `json:"unhealthy_count"`
}

// CheckHealth performs comprehensive health checks
func (h *EnhancedHealthChecker) CheckHealth(ctx context.Context) *EnhancedHealthStatus {
	start := time.Now()
	
	// Check all components
	dbHealth := h.checkDatabaseHealth(ctx)
	redisHealth := h.checkRedisHealth(ctx)
	eventHealth := h.checkEventSystemHealth(ctx)
	meshHealth := h.checkMeshNetworkHealth(ctx)
	systemHealth := h.getSystemHealth()

	// Calculate overall health
	overallHealthy := dbHealth.Healthy && redisHealth.Healthy && eventHealth.Healthy && meshHealth.Healthy
	status := "healthy"
	if !overallHealthy {
		status = "unhealthy"
	}
	
	// Calculate dependency health
	totalDeps := 4 // database, redis, event system, mesh network
	healthyDeps := 0
	if dbHealth.Healthy {
		healthyDeps++
	}
	if redisHealth.Healthy {
		healthyDeps++
	}
	if eventHealth.Healthy {
		healthyDeps++
	}
	if meshHealth.Healthy {
		healthyDeps++
	}
	
	dependencyHealth := &DependencyHealth{
		TotalDependencies: totalDeps,
		HealthyCount:      healthyDeps,
		UnhealthyCount:    totalDeps - healthyDeps,
	}
	
	responseTime := float64(time.Since(start).Nanoseconds()) / 1e6 // Convert to milliseconds
	
	return &EnhancedHealthStatus{
		Healthy:        overallHealthy,
		Status:         status,
		Version:        h.version,
		Timestamp:      time.Now().Unix(),
		Uptime:         h.getUptime(),
		ResponseTimeMs: responseTime,
		Details: &DetailedHealthStatus{
			Database:     dbHealth,
			Redis:        redisHealth,
			EventSystem:  eventHealth,
			MeshNetwork:  meshHealth,
			System:       systemHealth,
			Dependencies: dependencyHealth,
		},
	}
}

// checkDatabaseHealth performs database health checks
func (h *EnhancedHealthChecker) checkDatabaseHealth(ctx context.Context) *ComponentHealth {
	start := time.Now()
	
	// Test basic connectivity
	if err := h.db.PingContext(ctx); err != nil {
		return &ComponentHealth{
			Healthy:      false,
			Status:       "unhealthy",
			ResponseTime: float64(time.Since(start).Nanoseconds()) / 1e6,
			Error:        fmt.Sprintf("ping failed: %v", err),
		}
	}
	
	// Test query execution
	var count int
	query := "SELECT COUNT(*) FROM users"
	if err := h.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return &ComponentHealth{
			Healthy:      false,
			Status:       "unhealthy",
			ResponseTime: float64(time.Since(start).Nanoseconds()) / 1e6,
			Error:        fmt.Sprintf("query failed: %v", err),
		}
	}
	
	// Get connection stats
	stats := h.db.Stats()
	
	return &ComponentHealth{
		Healthy:      true,
		Status:       "healthy",
		ResponseTime: float64(time.Since(start).Nanoseconds()) / 1e6,
		Details: map[string]interface{}{
			"user_count":        count,
			"open_connections":  stats.OpenConnections,
			"in_use":           stats.InUse,
			"idle":             stats.Idle,
			"max_open_conns":   stats.MaxOpenConnections,
			"wait_count":       stats.WaitCount,
			"wait_duration_ms": float64(stats.WaitDuration.Nanoseconds()) / 1e6,
		},
	}
}

// checkRedisHealth performs Redis health checks
func (h *EnhancedHealthChecker) checkRedisHealth(ctx context.Context) *ComponentHealth {
	start := time.Now()
	
	// Test basic connectivity
	if err := h.redis.Ping(ctx).Err(); err != nil {
		return &ComponentHealth{
			Healthy:      false,
			Status:       "unhealthy",
			ResponseTime: float64(time.Since(start).Nanoseconds()) / 1e6,
			Error:        fmt.Sprintf("ping failed: %v", err),
		}
	}
	
	// Test read/write operations
	testKey := "health_check_test"
	testValue := fmt.Sprintf("test_%d", time.Now().Unix())
	
	// Set test value
	if err := h.redis.Set(ctx, testKey, testValue, time.Minute).Err(); err != nil {
		return &ComponentHealth{
			Healthy:      false,
			Status:       "unhealthy",
			ResponseTime: float64(time.Since(start).Nanoseconds()) / 1e6,
			Error:        fmt.Sprintf("set operation failed: %v", err),
		}
	}
	
	// Get test value
	result, err := h.redis.Get(ctx, testKey).Result()
	if err != nil || result != testValue {
		return &ComponentHealth{
			Healthy:      false,
			Status:       "unhealthy",
			ResponseTime: float64(time.Since(start).Nanoseconds()) / 1e6,
			Error:        fmt.Sprintf("get operation failed: %v", err),
		}
	}
	
	// Clean up test key
	h.redis.Del(ctx, testKey)
	
	// Get Redis info
	poolStats := h.redis.PoolStats()
	
	return &ComponentHealth{
		Healthy:      true,
		Status:       "healthy",
		ResponseTime: float64(time.Since(start).Nanoseconds()) / 1e6,
		Details: map[string]interface{}{
			"hits":         poolStats.Hits,
			"misses":       poolStats.Misses,
			"timeouts":     poolStats.Timeouts,
			"total_conns":  poolStats.TotalConns,
			"idle_conns":   poolStats.IdleConns,
			"stale_conns":  poolStats.StaleConns,
		},
	}
}

// checkEventSystemHealth performs event system health checks
func (h *EnhancedHealthChecker) checkEventSystemHealth(ctx context.Context) *ComponentHealth {
	start := time.Now()
	
	// Check event publisher
	if h.eventPublisher != nil {
		if err := h.eventPublisher.HealthCheck(); err != nil {
			return &ComponentHealth{
				Healthy:      false,
				Status:       "unhealthy",
				ResponseTime: float64(time.Since(start).Nanoseconds()) / 1e6,
				Error:        fmt.Sprintf("publisher health check failed: %v", err),
			}
		}
	}
	
	// Check event subscriber
	if h.eventSubscriber != nil {
		if err := h.eventSubscriber.HealthCheck(); err != nil {
			return &ComponentHealth{
				Healthy:      false,
				Status:       "unhealthy",
				ResponseTime: float64(time.Since(start).Nanoseconds()) / 1e6,
				Error:        fmt.Sprintf("subscriber health check failed: %v", err),
			}
		}
	}
	
	details := map[string]interface{}{
		"publisher_enabled":  h.eventPublisher != nil,
		"subscriber_enabled": h.eventSubscriber != nil,
	}
	
	if h.eventSubscriber != nil {
		details["subscribed_channels"] = h.eventSubscriber.GetSubscribedChannels()
	}
	
	return &ComponentHealth{
		Healthy:      true,
		Status:       "healthy",
		ResponseTime: float64(time.Since(start).Nanoseconds()) / 1e6,
		Details:      details,
	}
}

// checkMeshNetworkHealth performs mesh network health checks
func (h *EnhancedHealthChecker) checkMeshNetworkHealth(ctx context.Context) *ComponentHealth {
	start := time.Now()

	if h.meshClient == nil {
		return &ComponentHealth{
			Healthy:      false,
			Status:       "unhealthy",
			ResponseTime: float64(time.Since(start).Nanoseconds()) / 1e6,
			Error:        "mesh client not initialized",
		}
	}

	// Get mesh health information
	meshHealth := h.meshClient.HealthCheck()

	// Extract pool manager health
	poolManagerHealth, ok := meshHealth["pool_manager"].(map[string]interface{})
	if !ok {
		return &ComponentHealth{
			Healthy:      false,
			Status:       "unhealthy",
			ResponseTime: float64(time.Since(start).Nanoseconds()) / 1e6,
			Error:        "failed to get pool manager health",
		}
	}

	// Check if we have healthy connections
	totalPools, _ := poolManagerHealth["total_pools"].(int)
	healthyPools, _ := poolManagerHealth["healthy_pools"].(int)
	totalConnections, _ := poolManagerHealth["total_connections"].(int64)
	healthyConnections, _ := poolManagerHealth["healthy_connections"].(int64)

	// Consider mesh healthy if we have some healthy connections or no services to connect to
	meshHealthy := totalPools == 0 || (healthyPools > 0 && healthyConnections > 0)

	status := "healthy"
	if !meshHealthy {
		status = "unhealthy"
	}

	return &ComponentHealth{
		Healthy:      meshHealthy,
		Status:       status,
		ResponseTime: float64(time.Since(start).Nanoseconds()) / 1e6,
		Details: map[string]interface{}{
			"total_pools":        totalPools,
			"healthy_pools":      healthyPools,
			"total_connections":  totalConnections,
			"healthy_connections": healthyConnections,
			"pool_details":       poolManagerHealth["pool_details"],
		},
	}
}

// getSystemHealth returns system-level health metrics
func (h *EnhancedHealthChecker) getSystemHealth() *SystemHealth {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)
	
	memoryUsageMB := float64(memStats.Alloc) / 1024 / 1024
	
	return &SystemHealth{
		MemoryUsageMB:      memoryUsageMB,
		MemoryUsagePercent: (memoryUsageMB / 1024) * 100, // Rough estimate assuming 1GB available
		GoroutineCount:     runtime.NumGoroutine(),
		CPUCount:           runtime.NumCPU(),
	}
}

// getUptime returns the service uptime as a human-readable string
func (h *EnhancedHealthChecker) getUptime() string {
	uptime := time.Since(h.startTime)
	
	days := int(uptime.Hours()) / 24
	hours := int(uptime.Hours()) % 24
	minutes := int(uptime.Minutes()) % 60
	seconds := int(uptime.Seconds()) % 60
	
	if days > 0 {
		return fmt.Sprintf("%dd %dh %dm %ds", days, hours, minutes, seconds)
	} else if hours > 0 {
		return fmt.Sprintf("%dh %dm %ds", hours, minutes, seconds)
	} else if minutes > 0 {
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else {
		return fmt.Sprintf("%ds", seconds)
	}
}
