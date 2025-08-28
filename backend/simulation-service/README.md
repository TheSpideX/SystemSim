# Simulation Service

A sophisticated distributed systems simulation engine that provides realistic modeling of complex system architectures.

## Quick Start

### Prerequisites
- Go 1.23 or later
- Redis (for advanced features)

### Run the Service

```bash
# Navigate to the simulation service directory
cd backend/simulation-service

# Install dependencies
go mod tidy

# Run the service
go run cmd/server/main.go
```

The service will start on port 11000 by default.

### Test the Service

```bash
# Health check
curl http://localhost:11000/health

# Status check
curl http://localhost:11000/api/v1/status
```

## Configuration

Configuration is handled through environment variables. See `.env` file for available options.

Key configuration options:
- `PORT`: HTTP server port (default: 11000)
- `ENVIRONMENT`: Environment mode (development/production)

## Development Status

This is the initial setup phase. The service currently provides:
- ✅ Basic HTTP server with Gin
- ✅ Health check endpoints
- ✅ Environment configuration
- ✅ Go module setup

### Planned Features
- [ ] Base Engine System (CPU, Memory, Storage, Network)
- [ ] Registry and Coordination System
- [ ] ACID-like Isolation
- [ ] Backpressure and Health System
- [ ] Time Synchronization
- [ ] Decision Graphs and Routing
- [ ] gRPC Mesh Integration
- [ ] WebSocket Support

## Architecture

The simulation service is designed as a microservice that integrates with the larger system design simulator platform. It will provide:

1. **Realistic System Simulation**: Model real-world distributed systems with high accuracy
2. **Educational Value**: Teach distributed systems concepts through hands-on experience
3. **Production Patterns**: Implement patterns used by Netflix, Amazon, Google
4. **Scalability**: Handle thousands of components and complex interactions

## API Endpoints

### Health Endpoints
- `GET /health` - Service health check

### API Endpoints
- `GET /api/v1/status` - Service status

More endpoints will be added as features are implemented.
