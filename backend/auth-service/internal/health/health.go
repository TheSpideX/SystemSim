package health

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
)

// HealthStatus represents the overall health status
type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Version   string            `json:"version"`
	Uptime    string            `json:"uptime"`
	Checks    map[string]Check  `json:"checks"`
}

// Check represents an individual health check
type Check struct {
	Status   string        `json:"status"`
	Message  string        `json:"message,omitempty"`
	Latency  time.Duration `json:"latency_ms"`
	Details  interface{}   `json:"details,omitempty"`
}

// HealthChecker provides health checking functionality
type HealthChecker struct {
	db          *sql.DB
	redisClient *redis.Client
	startTime   time.Time
	version     string
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(db *sql.DB, redisClient *redis.Client, version string) *HealthChecker {
	return &HealthChecker{
		db:          db,
		redisClient: redisClient,
		startTime:   time.Now(),
		version:     version,
	}
}

// CheckLiveness performs a basic liveness check
func (h *HealthChecker) CheckLiveness() *HealthStatus {
	return &HealthStatus{
		Status:    "healthy",
		Timestamp: time.Now(),
		Version:   h.version,
		Uptime:    time.Since(h.startTime).String(),
		Checks: map[string]Check{
			"service": {
				Status:  "healthy",
				Message: "Service is running",
				Latency: 0,
			},
		},
	}
}

// CheckReadiness performs readiness checks for all dependencies
func (h *HealthChecker) CheckReadiness() *HealthStatus {
	checks := make(map[string]Check)
	overallStatus := "healthy"

	// Check database
	dbCheck := h.checkDatabase()
	checks["database"] = dbCheck
	if dbCheck.Status != "healthy" {
		overallStatus = "unhealthy"
	}

	// Check Redis
	redisCheck := h.checkRedis()
	checks["redis"] = redisCheck
	if redisCheck.Status != "healthy" {
		overallStatus = "unhealthy"
	}

	return &HealthStatus{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Version:   h.version,
		Uptime:    time.Since(h.startTime).String(),
		Checks:    checks,
	}
}

// CheckDetailed performs comprehensive health checks
func (h *HealthChecker) CheckDetailed() *HealthStatus {
	checks := make(map[string]Check)
	overallStatus := "healthy"

	// Check database
	dbCheck := h.checkDatabase()
	checks["database"] = dbCheck
	if dbCheck.Status != "healthy" {
		overallStatus = "unhealthy"
	}

	// Check Redis
	redisCheck := h.checkRedis()
	checks["redis"] = redisCheck
	if redisCheck.Status != "healthy" {
		overallStatus = "unhealthy"
	}

	// Check system resources
	memoryCheck := h.checkMemory()
	checks["memory"] = memoryCheck
	if memoryCheck.Status != "healthy" {
		overallStatus = "degraded"
	}

	// Check disk space
	diskCheck := h.checkDisk()
	checks["disk"] = diskCheck
	if diskCheck.Status != "healthy" {
		overallStatus = "degraded"
	}

	return &HealthStatus{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Version:   h.version,
		Uptime:    time.Since(h.startTime).String(),
		Checks:    checks,
	}
}

// checkDatabase checks database connectivity and performance
func (h *HealthChecker) checkDatabase() Check {
	start := time.Now()
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test connection
	if err := h.db.PingContext(ctx); err != nil {
		return Check{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Database ping failed: %v", err),
			Latency: time.Since(start),
		}
	}

	// Test query performance
	var count int
	err := h.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return Check{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Database query failed: %v", err),
			Latency: time.Since(start),
		}
	}

	latency := time.Since(start)
	status := "healthy"
	message := "Database is responsive"

	// Check if query is slow
	if latency > 1*time.Second {
		status = "degraded"
		message = "Database is slow"
	}

	return Check{
		Status:  status,
		Message: message,
		Latency: latency,
		Details: map[string]interface{}{
			"user_count": count,
		},
	}
}

// checkRedis checks Redis connectivity and performance
func (h *HealthChecker) checkRedis() Check {
	start := time.Now()
	
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Test ping
	if err := h.redisClient.Ping(ctx).Err(); err != nil {
		return Check{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Redis ping failed: %v", err),
			Latency: time.Since(start),
		}
	}

	// Test set/get operation
	testKey := "health_check_test"
	testValue := "test_value"
	
	if err := h.redisClient.Set(ctx, testKey, testValue, time.Minute).Err(); err != nil {
		return Check{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Redis set failed: %v", err),
			Latency: time.Since(start),
		}
	}

	val, err := h.redisClient.Get(ctx, testKey).Result()
	if err != nil || val != testValue {
		return Check{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Redis get failed: %v", err),
			Latency: time.Since(start),
		}
	}

	// Clean up test key
	h.redisClient.Del(ctx, testKey)

	latency := time.Since(start)
	status := "healthy"
	message := "Redis is responsive"

	if latency > 500*time.Millisecond {
		status = "degraded"
		message = "Redis is slow"
	}

	return Check{
		Status:  status,
		Message: message,
		Latency: latency,
	}
}

// checkMemory checks system memory usage
func (h *HealthChecker) checkMemory() Check {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Convert bytes to MB
	allocMB := m.Alloc / 1024 / 1024
	sysMB := m.Sys / 1024 / 1024

	status := "healthy"
	message := "Memory usage is normal"

	// Check if memory usage is high (>500MB allocated)
	if allocMB > 500 {
		status = "degraded"
		message = "High memory usage"
	}

	return Check{
		Status:  status,
		Message: message,
		Latency: 0,
		Details: map[string]interface{}{
			"alloc_mb":     allocMB,
			"sys_mb":       sysMB,
			"num_gc":       m.NumGC,
			"goroutines":   runtime.NumGoroutine(),
		},
	}
}

// checkDisk checks available disk space
func (h *HealthChecker) checkDisk() Check {
	var stat syscall.Statfs_t
	err := syscall.Statfs("/", &stat)
	if err != nil {
		return Check{
			Status:  "unhealthy",
			Message: fmt.Sprintf("Failed to get disk stats: %v", err),
			Latency: 0,
		}
	}

	// Calculate available space in GB
	availableGB := (stat.Bavail * uint64(stat.Bsize)) / (1024 * 1024 * 1024)
	totalGB := (stat.Blocks * uint64(stat.Bsize)) / (1024 * 1024 * 1024)
	usedPercent := float64(stat.Blocks-stat.Bavail) / float64(stat.Blocks) * 100

	status := "healthy"
	message := "Disk space is sufficient"

	// Check if disk usage is high (>90%)
	if usedPercent > 90 {
		status = "degraded"
		message = "Low disk space"
	}

	return Check{
		Status:  status,
		Message: message,
		Latency: 0,
		Details: map[string]interface{}{
			"available_gb": availableGB,
			"total_gb":     totalGB,
			"used_percent": usedPercent,
		},
	}
}
