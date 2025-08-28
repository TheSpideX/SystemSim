package metrics

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// Counter represents a simple counter metric
type Counter struct {
	value int64
	mu    sync.RWMutex
}

// Gauge represents a gauge metric
type Gauge struct {
	value float64
	mu    sync.RWMutex
}

// Histogram represents a histogram metric
type Histogram struct {
	buckets map[float64]int64
	sum     float64
	count   int64
	mu      sync.RWMutex
}

// MetricsCollector collects and manages metrics
type MetricsCollector struct {
	counters   map[string]*Counter
	gauges     map[string]*Gauge
	histograms map[string]*Histogram
	mu         sync.RWMutex
	startTime  time.Time
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		histograms: make(map[string]*Histogram),
		startTime:  time.Now(),
	}
}

// Global metrics collector instance
var globalCollector = NewMetricsCollector()

// GetCollector returns the global metrics collector
func GetCollector() *MetricsCollector {
	return globalCollector
}

// Counter methods
func (c *Counter) Inc() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value++
}

func (c *Counter) Add(delta int64) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.value += delta
}

func (c *Counter) Value() int64 {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.value
}

// Gauge methods
func (g *Gauge) Set(value float64) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value = value
}

func (g *Gauge) Inc() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value++
}

func (g *Gauge) Dec() {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.value--
}

func (g *Gauge) Value() float64 {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.value
}

// Histogram methods
func (h *Histogram) Observe(value float64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	
	h.sum += value
	h.count++
	
	// Update buckets
	for bucket := range h.buckets {
		if value <= bucket {
			h.buckets[bucket]++
		}
	}
}

func (h *Histogram) Count() int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.count
}

func (h *Histogram) Sum() float64 {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.sum
}

// MetricsCollector methods
func (m *MetricsCollector) GetOrCreateCounter(name string) *Counter {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if counter, exists := m.counters[name]; exists {
		return counter
	}
	
	counter := &Counter{}
	m.counters[name] = counter
	return counter
}

func (m *MetricsCollector) GetOrCreateGauge(name string) *Gauge {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if gauge, exists := m.gauges[name]; exists {
		return gauge
	}
	
	gauge := &Gauge{}
	m.gauges[name] = gauge
	return gauge
}

func (m *MetricsCollector) GetOrCreateHistogram(name string, buckets []float64) *Histogram {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if histogram, exists := m.histograms[name]; exists {
		return histogram
	}
	
	bucketMap := make(map[float64]int64)
	for _, bucket := range buckets {
		bucketMap[bucket] = 0
	}
	
	histogram := &Histogram{
		buckets: bucketMap,
	}
	m.histograms[name] = histogram
	return histogram
}

// Prometheus-style metrics output
func (m *MetricsCollector) PrometheusFormat() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var output string
	
	// Add uptime
	uptime := time.Since(m.startTime).Seconds()
	output += fmt.Sprintf("# HELP auth_service_uptime_seconds Time since service started\n")
	output += fmt.Sprintf("# TYPE auth_service_uptime_seconds gauge\n")
	output += fmt.Sprintf("auth_service_uptime_seconds %.2f\n\n", uptime)
	
	// Add counters
	for name, counter := range m.counters {
		output += fmt.Sprintf("# HELP %s Counter metric\n", name)
		output += fmt.Sprintf("# TYPE %s counter\n", name)
		output += fmt.Sprintf("%s %d\n\n", name, counter.Value())
	}
	
	// Add gauges
	for name, gauge := range m.gauges {
		output += fmt.Sprintf("# HELP %s Gauge metric\n", name)
		output += fmt.Sprintf("# TYPE %s gauge\n", name)
		output += fmt.Sprintf("%s %.2f\n\n", name, gauge.Value())
	}
	
	// Add histograms
	for name, histogram := range m.histograms {
		output += fmt.Sprintf("# HELP %s Histogram metric\n", name)
		output += fmt.Sprintf("# TYPE %s histogram\n", name)
		
		// Add buckets
		for bucket, count := range histogram.buckets {
			output += fmt.Sprintf("%s_bucket{le=\"%.2f\"} %d\n", name, bucket, count)
		}
		
		// Add sum and count
		output += fmt.Sprintf("%s_sum %.2f\n", name, histogram.Sum())
		output += fmt.Sprintf("%s_count %d\n\n", name, histogram.Count())
	}
	
	return output
}

// Initialize default metrics
func InitializeMetrics() {
	collector := GetCollector()
	
	// Authentication metrics
	collector.GetOrCreateCounter("auth_login_attempts_total")
	collector.GetOrCreateCounter("auth_login_success_total")
	collector.GetOrCreateCounter("auth_login_failures_total")
	collector.GetOrCreateCounter("auth_registrations_total")
	collector.GetOrCreateCounter("auth_token_refreshes_total")
	
	// Session metrics
	collector.GetOrCreateGauge("auth_active_sessions")
	collector.GetOrCreateCounter("auth_sessions_created_total")
	collector.GetOrCreateCounter("auth_sessions_revoked_total")
	
	// HTTP metrics
	collector.GetOrCreateCounter("http_requests_total")
	collector.GetOrCreateHistogram("http_request_duration_seconds", []float64{0.1, 0.5, 1.0, 2.0, 5.0})
	
	// Error metrics
	collector.GetOrCreateCounter("auth_errors_total")
	collector.GetOrCreateCounter("database_errors_total")
	collector.GetOrCreateCounter("redis_errors_total")
}

// Middleware for collecting HTTP metrics
func HTTPMetricsMiddleware() gin.HandlerFunc {
	collector := GetCollector()
	requestsTotal := collector.GetOrCreateCounter("http_requests_total")
	requestDuration := collector.GetOrCreateHistogram("http_request_duration_seconds", []float64{0.1, 0.5, 1.0, 2.0, 5.0})
	
	return func(c *gin.Context) {
		start := time.Now()
		
		c.Next()
		
		// Record metrics
		requestsTotal.Inc()
		duration := time.Since(start).Seconds()
		requestDuration.Observe(duration)
	}
}

// Helper functions for common metrics
func IncrementLoginAttempts() {
	GetCollector().GetOrCreateCounter("auth_login_attempts_total").Inc()
}

func IncrementLoginSuccess() {
	GetCollector().GetOrCreateCounter("auth_login_success_total").Inc()
}

func IncrementLoginFailures() {
	GetCollector().GetOrCreateCounter("auth_login_failures_total").Inc()
}

func IncrementRegistrations() {
	GetCollector().GetOrCreateCounter("auth_registrations_total").Inc()
}

func IncrementTokenRefreshes() {
	GetCollector().GetOrCreateCounter("auth_token_refreshes_total").Inc()
}

func SetActiveSessions(count int64) {
	GetCollector().GetOrCreateGauge("auth_active_sessions").Set(float64(count))
}

func IncrementSessionsCreated() {
	GetCollector().GetOrCreateCounter("auth_sessions_created_total").Inc()
}

func IncrementSessionsRevoked() {
	GetCollector().GetOrCreateCounter("auth_sessions_revoked_total").Inc()
}

func IncrementAuthErrors() {
	GetCollector().GetOrCreateCounter("auth_errors_total").Inc()
}

func IncrementDatabaseErrors() {
	GetCollector().GetOrCreateCounter("database_errors_total").Inc()
}

func IncrementRedisErrors() {
	GetCollector().GetOrCreateCounter("redis_errors_total").Inc()
}

// MetricsHandler returns metrics in Prometheus format
func MetricsHandler(c *gin.Context) {
	c.Header("Content-Type", "text/plain; charset=utf-8")
	c.String(http.StatusOK, GetCollector().PrometheusFormat())
}
