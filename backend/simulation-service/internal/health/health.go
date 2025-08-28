package health

import (
	"fmt"
	"runtime"
	"sync"
	"time"

	"github.com/systemsim/simulation-service/internal/database"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
)

// HealthCheck represents a single health check
type HealthCheck struct {
	Name        string                 `json:"name"`
	Status      HealthStatus           `json:"status"`
	Message     string                 `json:"message"`
	LastChecked time.Time              `json:"last_checked"`
	Duration    time.Duration          `json:"duration"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// HealthReport represents the overall health report
type HealthReport struct {
	Status      HealthStatus           `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	Version     string                 `json:"version"`
	Uptime      time.Duration          `json:"uptime"`
	Checks      []HealthCheck          `json:"checks"`
	SystemInfo  map[string]interface{} `json:"system_info"`
}

// HealthChecker manages health checks for the simulation service
type HealthChecker struct {
	redisClient *database.RedisClient
	startTime   time.Time
	mu          sync.RWMutex
	checks      map[string]HealthCheck
}

// NewHealthChecker creates a new health checker
func NewHealthChecker(redisClient *database.RedisClient) *HealthChecker {
	return &HealthChecker{
		redisClient: redisClient,
		startTime:   time.Now(),
		checks:      make(map[string]HealthCheck),
	}
}

// CheckHealth performs all health checks and returns a report
func (h *HealthChecker) CheckHealth() *HealthReport {
	h.mu.Lock()
	defer h.mu.Unlock()

	checks := []HealthCheck{
		h.checkRedis(),
		h.checkMemory(),
		h.checkGoroutines(),
	}

	// Determine overall status
	overallStatus := HealthStatusHealthy
	for _, check := range checks {
		if check.Status == HealthStatusUnhealthy {
			overallStatus = HealthStatusUnhealthy
			break
		} else if check.Status == HealthStatusDegraded && overallStatus == HealthStatusHealthy {
			overallStatus = HealthStatusDegraded
		}
	}

	return &HealthReport{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Version:    "1.0.0",
		Uptime:     time.Since(h.startTime),
		Checks:     checks,
		SystemInfo: h.getSystemInfo(),
	}
}

// CheckReadiness checks if the service is ready to serve requests
func (h *HealthChecker) CheckReadiness() *HealthReport {
	h.mu.Lock()
	defer h.mu.Unlock()

	checks := []HealthCheck{
		h.checkRedis(),
	}

	// For readiness, all checks must be healthy
	overallStatus := HealthStatusHealthy
	for _, check := range checks {
		if check.Status != HealthStatusHealthy {
			overallStatus = HealthStatusUnhealthy
			break
		}
	}

	return &HealthReport{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Version:    "1.0.0",
		Uptime:     time.Since(h.startTime),
		Checks:     checks,
		SystemInfo: h.getSystemInfo(),
	}
}

// CheckLiveness checks if the service is alive
func (h *HealthChecker) CheckLiveness() *HealthReport {
	return &HealthReport{
		Status:     HealthStatusHealthy,
		Timestamp:  time.Now(),
		Version:    "1.0.0",
		Uptime:     time.Since(h.startTime),
		Checks:     []HealthCheck{},
		SystemInfo: h.getSystemInfo(),
	}
}

// checkRedis checks Redis connectivity
func (h *HealthChecker) checkRedis() HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Name:        "redis",
		LastChecked: start,
	}

	if err := h.redisClient.Ping(); err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("Redis connection failed: %v", err)
	} else {
		check.Status = HealthStatusHealthy
		check.Message = "Redis connection healthy"
	}

	check.Duration = time.Since(start)
	return check
}

// checkMemory checks memory usage
func (h *HealthChecker) checkMemory() HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Name:        "memory",
		LastChecked: start,
	}

	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	// Convert bytes to MB
	allocMB := float64(m.Alloc) / 1024 / 1024
	sysMB := float64(m.Sys) / 1024 / 1024

	check.Details = map[string]interface{}{
		"alloc_mb":     allocMB,
		"sys_mb":       sysMB,
		"num_gc":       m.NumGC,
		"gc_cpu_frac":  m.GCCPUFraction,
	}

	// Memory thresholds (in MB)
	if allocMB > 1000 {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("High memory usage: %.2f MB", allocMB)
	} else if allocMB > 500 {
		check.Status = HealthStatusDegraded
		check.Message = fmt.Sprintf("Elevated memory usage: %.2f MB", allocMB)
	} else {
		check.Status = HealthStatusHealthy
		check.Message = fmt.Sprintf("Memory usage normal: %.2f MB", allocMB)
	}

	check.Duration = time.Since(start)
	return check
}

// checkGoroutines checks goroutine count
func (h *HealthChecker) checkGoroutines() HealthCheck {
	start := time.Now()
	check := HealthCheck{
		Name:        "goroutines",
		LastChecked: start,
	}

	numGoroutines := runtime.NumGoroutine()
	check.Details = map[string]interface{}{
		"count": numGoroutines,
	}

	// Goroutine thresholds
	if numGoroutines > 10000 {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("Too many goroutines: %d", numGoroutines)
	} else if numGoroutines > 1000 {
		check.Status = HealthStatusDegraded
		check.Message = fmt.Sprintf("High goroutine count: %d", numGoroutines)
	} else {
		check.Status = HealthStatusHealthy
		check.Message = fmt.Sprintf("Goroutine count normal: %d", numGoroutines)
	}

	check.Duration = time.Since(start)
	return check
}

// getSystemInfo returns system information
func (h *HealthChecker) getSystemInfo() map[string]interface{} {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	return map[string]interface{}{
		"go_version":     runtime.Version(),
		"go_os":          runtime.GOOS,
		"go_arch":        runtime.GOARCH,
		"cpu_count":      runtime.NumCPU(),
		"goroutines":     runtime.NumGoroutine(),
		"memory_alloc":   m.Alloc,
		"memory_sys":     m.Sys,
		"gc_runs":        m.NumGC,
		"uptime_seconds": time.Since(h.startTime).Seconds(),
	}
}
