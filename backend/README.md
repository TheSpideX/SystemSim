# System Design Simulator - Microservice Backend

A production-ready microservice mesh architecture for system design simulation, featuring gRPC mesh networking, real-time communication, and comprehensive authentication services.

## 🏗️ **Architecture Overview**

This backend implements a **microservice mesh network** with three communication channels per service:
- **gRPC Mesh**: High-performance inter-service communication (20 connections per service)
- **HTTP + Async Queue**: Non-blocking operations and heavy processing
- **Redis Pub/Sub**: Real-time data streaming to simulation engine

### **Current Implementation Status**
- ✅ **Auth Service**: Production-ready with JWT, PostgreSQL, Redis
- 🔄 **Project Service**: Planned (Port 10000-10020)
- 🔄 **Simulation Service**: Planned (Port 11000-11020)
- 🔄 **Server Service (API Gateway)**: Planned (Port 8000-8020)

## 🚀 **Microservice Features**

### **✅ Auth Service (Production Ready)**
- **JWT Authentication**: Access + refresh tokens with secure handling
- **User Management**: Registration, login, profile management
- **Security Features**: Account lockout, rate limiting, password strength validation
- **Session Management**: Redis-backed sessions with expiration
- **Email System**: Verification and password reset functionality
- **Database**: PostgreSQL with proper indexing and connection pooling
- **Containerization**: Docker with health checks and graceful shutdown

### **🔄 Planned Services**
- **Project Service**: CRUD operations, templates, sharing, version control
- **Simulation Service**: Real-time simulation processing, performance calculations
- **Server Service**: API Gateway with WebSocket support and load balancing

### **🌐 Communication Architecture**
- **gRPC Mesh**: 20 connections per service for high-performance RPC
- **HTTP + Queue**: Async processing with message brokers
- **Redis Pub/Sub**: Real-time streaming for simulation data
- **WebSocket**: Client real-time connections and notifications

## 📁 **Project Structure**

```
backend/
├── auth-service/                        # ✅ Production-ready auth microservice
│   ├── cmd/server/main.go              # Service entry point
│   ├── internal/
│   │   ├── config/config.go            # Configuration management
│   │   ├── database/                   # Database connections (PostgreSQL + Redis)
│   │   ├── handlers/                   # HTTP request handlers
│   │   ├── middleware/                 # Security and auth middleware
│   │   ├── models/                     # Data models and DTOs
│   │   ├── repository/                 # Data access layer
│   │   ├── security/                   # JWT and password security
│   │   └── services/                   # Business logic layer
│   ├── migrations/                     # Database migration files
│   ├── Dockerfile                      # Container configuration
│   ├── docker-compose.yml              # Local development setup
│   └── go.mod                          # Service dependencies
├── cmd/                                # 🔄 Legacy simulation engine (being refactored)
│   ├── server/main.go                  # Current HTTP server
│   └── simulator/                      # Simulation components
├── internal/                           # 🔄 Shared simulation engine components
│   ├── api/                           # API handlers
│   ├── components/                     # System components
│   ├── engines/                       # Simulation engines
│   └── simulation/                    # Core simulation logic
├── scripts/                           # Build and deployment scripts
│   ├── build.sh                       # Build automation
│   └── run_tests.sh                   # Test runner
├── docs/                              # 📚 Architecture documentation
│   ├── microservice-mesh-architecture.md  # Current architecture
│   ├── implementation-roadmap.md          # Implementation plan
│   └── system-architecture.md             # Legacy architecture
├── go.mod                             # Root module definition
└── README.md                          # This file
```

## 🛠 **Installation & Setup**

### **Prerequisites**
- Go 1.21 or later
- Docker & Docker Compose
- PostgreSQL 15+
- Redis 7+

### **Quick Start - Auth Service**
```bash
# Clone the repository
git clone <repository-url>
cd backend/auth-service

# Start dependencies (PostgreSQL + Redis)
docker-compose up -d postgres redis

# Install dependencies
go mod tidy

# Run database migrations
go run cmd/server/main.go # Migrations run automatically

# Start the auth service
go run cmd/server/main.go
```

### **Docker Development Environment**
```bash
# Start complete auth service stack
cd backend/auth-service
docker-compose up -d

# Auth service will be available at:
# HTTP API: http://localhost:8001
# Health check: http://localhost:8001/health
```

## 🚀 **API Usage Examples**

### **Authentication Flow**
```bash
# Register a new user
curl -X POST http://localhost:8001/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!",
    "first_name": "John",
    "last_name": "Doe",
    "company": "Tech Corp"
  }'

# Login
curl -X POST http://localhost:8001/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePass123!"
  }'

# Response includes access_token and refresh_token
{
  "user": {
    "id": "uuid",
    "email": "user@example.com",
    "first_name": "John",
    "last_name": "Doe"
  },
  "access_token": "eyJhbGciOiJIUzI1NiIs...",
  "refresh_token": "eyJhbGciOiJIUzI1NiIs...",
  "expires_in": 900
}
```

### **Protected Endpoints**
```bash
# Get user profile (requires Authorization header)
curl -X GET http://localhost:8001/api/v1/user/profile \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"

# Update profile
curl -X PUT http://localhost:8001/api/v1/user/profile \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "Jane",
    "company": "New Company"
  }'
```

### Using Engine Factory
```go
// Create factory
factory := engines.NewEngineFactory()

// Configure engine
config := engines.EngineConfiguration{
    ID:      "my-web-server",
    Type:    engines.ComputeEngine,
    Profile: "web_server",
    Specs: engines.RealWorldSpecs{
        CPUCores:        4,
        CPUFrequencyGHz: 2.4,
        MemoryGB:        8.0,
        StorageGB:       100.0,
        NetworkMbps:     1000.0,
        MinLatency:      time.Millisecond,
        MaxThroughput:   15000.0,
    },
    CustomConfig: map[string]interface{}{
        "monitoring_enabled": true,
        "chaos_level":        0.1,
    },
}

// Create engine
engine, err := factory.CreateEngine(config, messageBus)
if err != nil {
    log.Fatal(err)
}
```

### Using Templates
```go
// Get a predefined template
template, exists := engines.GetTemplateByName("high_performance_web_server")
if !exists {
    log.Fatal("Template not found")
}

// Create engine from template
engine, err := factory.CreateEngineFromTemplate(*template, "my-server", messageBus)
if err != nil {
    log.Fatal(err)
}
```

## 🔧 Configuration

### Engine Profiles
Engines can be configured with predefined profiles that set realistic performance characteristics:

```go
// Available compute profiles
profiles := []string{
    "web_server", "api_server", "application_server",
    "microservice", "batch_processor", "real_time_processor",
}

// Set profile
engine.SetProfile("web_server")
```

### Monitoring & Alerting
```go
// Enable monitoring
engine.EnableMonitoring()

// Get metrics
metrics := engine.GetAllMetrics()
alerts := engine.GetActiveAlerts()

// Custom alert rule
rule := engines.AlertRule{
    ID:          "high_cpu",
    Name:        "High CPU Usage",
    MetricName:  "cpu_usage",
    Threshold:   85.0,
    Duration:    2 * time.Minute,
    Severity:    "high",
}
engine.AddCustomAlertRule(rule)
```

### Failure Injection
```go
// Enable chaos engineering
engine.EnableChaosEngineering(0.1) // 10% chaos level

// Get failure statistics
stats := engine.GetFailureStatistics()
fmt.Printf("Total failures: %d\n", stats.TotalFailures)
```

### Statistical Analysis
```go
// Enable statistical analysis
engine.EnableStatisticalAnalysis()

// Get analysis results
results := engine.GetAnalysisResults(10)
anomalies := engine.GetDetectedAnomalies(false)
trends := engine.GetTrendResults()
forecasts := engine.GetCapacityForecasts()
```

## 🧪 Testing

### Run All Tests
```bash
# Run comprehensive test suite
./scripts/run_tests.sh

# Run with coverage report
./scripts/run_tests.sh --open-coverage

# Run with cleanup
./scripts/run_tests.sh --cleanup
```

### Run Specific Test Types
```bash
# Unit tests only
go test -v ./internal/engines/

# Integration tests
go test -v -tags=integration ./internal/engines/

# Benchmarks
go test -v -bench=. -benchtime=30s ./internal/engines/

# Race condition detection
go test -v -race ./internal/engines/

# Performance validation
go test -v -run="TestPerformanceValidation" ./internal/engines/
```

## 📊 Performance Benchmarks

The system includes comprehensive performance validation based on real-world benchmarks:

### Compute Engines
- **Web Server**: 15,000 req/s, 2ms avg latency
- **API Server**: 8,000 req/s, 5ms avg latency
- **Microservice**: 5,000 req/s, 3ms avg latency

### Storage Engines
- **PostgreSQL**: 5,000 queries/s, 3ms avg latency
- **Redis**: 100,000 ops/s, 0.1ms avg latency
- **MongoDB**: 8,000 ops/s, 4ms avg latency

### Network Engines
- **NGINX Load Balancer**: 50,000 req/s, 0.5ms avg latency
- **API Gateway**: 20,000 req/s, 2ms avg latency
- **CDN**: 200,000 req/s, 0.1ms avg latency

## 🔍 Monitoring & Observability

### Available Metrics
- **Performance**: CPU usage, memory usage, latency, throughput
- **Reliability**: Error rate, success rate, availability
- **Resource**: Queue length, active connections, processing count
- **Custom**: Engine-specific metrics based on type and profile

### Alerting
- **Threshold-based**: CPU > 85%, Memory > 90%, Error rate > 5%
- **Anomaly detection**: Statistical anomaly detection with Z-score and IQR
- **Trend analysis**: Performance trend monitoring and forecasting
- **Capacity planning**: Resource exhaustion prediction

### Dashboards
- **System Overview**: High-level system health and performance
- **Engine Details**: Detailed metrics for individual engines
- **Custom Dashboards**: User-defined monitoring dashboards

## 🎯 Use Cases

### Educational
- **System Design Learning**: Understand how different components interact
- **Performance Analysis**: Learn about system bottlenecks and optimization
- **Failure Scenarios**: Experience how systems behave under failure conditions

### Professional
- **Architecture Planning**: Model and validate system architectures
- **Capacity Planning**: Predict resource requirements and scaling needs
- **Disaster Recovery**: Test system resilience and recovery procedures
- **Performance Optimization**: Identify optimization opportunities

### Research
- **Algorithm Comparison**: Compare different algorithms and approaches
- **Scaling Studies**: Research system scaling patterns and limits
- **Failure Analysis**: Study failure propagation and recovery patterns

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

### Development Guidelines
- Write comprehensive tests for new features
- Follow Go best practices and conventions
- Update documentation for API changes
- Ensure all tests pass before submitting PR

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 🙏 Acknowledgments

- Performance benchmarks based on real-world systems and public benchmarks
- Failure scenarios inspired by actual production incidents
- Monitoring patterns based on industry best practices
- Statistical methods from academic research in system performance analysis
