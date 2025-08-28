# SystemSim Auth Service

A robust, production-ready authentication microservice for the SystemSim platform.

## Features

### 🔐 Security
- **TLS Encryption** with HTTP/2 for secure communication
- **JWT Authentication** with access and refresh tokens
- **Password Security** with bcrypt hashing and strength validation
- **Rate Limiting** to prevent brute force attacks
- **Account Lockout** after failed login attempts
- **Security Headers** (CORS, XSS protection, HSTS, etc.)
- **Input Validation** and sanitization

### 🚀 Performance
- **HTTP/2 Support** with TLS for improved performance and multiplexing
- **Redis Caching** for session management
- **Connection Pooling** for database connections
- **Optimized Queries** with proper indexing
- **Graceful Shutdown** with connection cleanup

### 📊 Monitoring
- **Health Checks** for service monitoring
- **Request Logging** with structured format
- **Error Handling** with proper HTTP status codes
- **Metrics Ready** for Prometheus integration

## API Endpoints

### Public Endpoints
- `POST /api/v1/auth/register` - User registration
- `POST /api/v1/auth/login` - User login
- `POST /api/v1/auth/refresh` - Refresh access token
- `POST /api/v1/auth/forgot-password` - Request password reset
- `POST /api/v1/auth/reset-password` - Reset password with token
- `POST /api/v1/auth/verify-email` - Verify email address
- `GET /health` - Health check

### Protected Endpoints (Require Authentication)
- `POST /api/v1/auth/logout` - User logout
- `GET /api/v1/user/profile` - Get user profile
- `PUT /api/v1/user/profile` - Update user profile
- `POST /api/v1/user/change-password` - Change password
- `DELETE /api/v1/user/account` - Delete user account

## Quick Start

### Prerequisites
- Go 1.21+
- PostgreSQL 15+
- Redis 7+
- Docker & Docker Compose (optional)

### Environment Setup

1. Copy environment file:
```bash
cp .env.example .env
```

2. Update the `.env` file with your configuration:
```bash
# Required: Change the JWT secret
JWT_SECRET=your-super-secret-jwt-key-change-this-in-production-must-be-at-least-32-characters

# Database connection
DATABASE_URL=postgres://auth_user:auth_password@localhost:5432/systemsim_auth?sslmode=disable

# Redis connection
REDIS_ADDR=localhost:6379
```

### Running with Docker Compose (Recommended)

```bash
# Start all services (PostgreSQL, Redis, Auth Service)
docker-compose up -d

# View logs
docker-compose logs -f auth-service

# Stop services
docker-compose down
```

### Running Locally

1. Start PostgreSQL and Redis:
```bash
# Using Docker
docker run -d --name postgres -p 5432:5432 -e POSTGRES_DB=systemsim_auth -e POSTGRES_USER=auth_user -e POSTGRES_PASSWORD=auth_password postgres:15-alpine
docker run -d --name redis -p 6379:6379 redis:7-alpine
```

2. Install dependencies:
```bash
go mod download
```

3. Run migrations:
```bash
# Migrations run automatically on startup
```

4. Start the service:
```bash
go run cmd/server/main.go
```

## Usage Examples

### Register a new user
```bash
curl -k -X POST https://localhost:9001/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123!",
    "first_name": "John",
    "last_name": "Doe",
    "company": "Example Corp"
  }'
```

### Login
```bash
curl -k -X POST https://localhost:9001/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123!"
  }'
```

### Access protected endpoint
```bash
curl -k -X GET https://localhost:9001/api/v1/user/profile \
  -H "Authorization: Bearer YOUR_ACCESS_TOKEN"
```

> **Note**: The `-k` flag is used to skip certificate verification for self-signed certificates in development. In production, use proper certificates and remove this flag.

## Configuration

### Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `HTTP_PORT` | HTTP/2 server port | `9001` |
| `GRPC_PORT` | gRPC server port | `9000` |
| `GIN_MODE` | Gin mode (development/production) | `development` |
| `DATABASE_URL` | PostgreSQL connection string | Required |
| `REDIS_ADDR` | Redis address | `localhost:6379` |
| `JWT_SECRET` | JWT signing secret (min 32 chars) | Required |
| `JWT_ACCESS_DURATION` | Access token duration | `15m` |
| `JWT_REFRESH_DURATION` | Refresh token duration | `168h` |
| `RATE_LIMIT_RPM` | Requests per minute limit | `60` |
| `HTTP2_ENABLED` | Enable HTTP/2 protocol | `true` |
| `TLS_ENABLED` | Enable TLS encryption | `true` |
| `TLS_CERT_FILE` | TLS certificate file path | `certs/server.crt` |
| `TLS_KEY_FILE` | TLS private key file path | `certs/server.key` |
| `TLS_MIN_VERSION` | Minimum TLS version (1.2/1.3) | `1.2` |

### Security Configuration

- **TLS Encryption**: HTTP/2 with TLS 1.2/1.3 support
- **Certificate Management**: Auto-generated self-signed certificates for development
- **Password Requirements**: Minimum 8 characters, must contain uppercase, lowercase, digit, and special character
- **Account Lockout**: 5 failed attempts locks account for 15 minutes
- **Rate Limiting**: 60 requests per minute per IP/user
- **Login Rate Limiting**: 5 login attempts per 15 minutes
- **Session Management**: Redis-backed with automatic cleanup
- **Security Headers**: HSTS, CORS, XSS protection, and content security policy

## Database Schema

### Users Table
- Comprehensive user information with security fields
- Email verification and password reset tokens
- Failed login attempt tracking and account lockout
- JSONB fields for flexible preferences storage

### Sessions Table
- JWT token management with refresh tokens
- Device and IP tracking for security
- Automatic cleanup of expired sessions

## Testing

```bash
# Run unit tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run integration tests
go test -tags=integration ./...
```

## Production Deployment

### Security Checklist
- [ ] Change default JWT secret
- [ ] Use strong database passwords
- [ ] Enable SSL/TLS for database connections
- [ ] Configure proper CORS origins
- [ ] Set up monitoring and alerting
- [ ] Enable audit logging
- [ ] Configure backup strategy

### Performance Tuning
- [ ] Optimize database connection pool settings
- [ ] Configure Redis memory limits
- [ ] Set up database read replicas if needed
- [ ] Configure load balancing
- [ ] Enable HTTP/2 and compression

## Monitoring

### Health Checks
- `GET /health` - Service health status
- Database connectivity check
- Redis connectivity check

### Metrics (Ready for Prometheus)
- Request duration and count
- Authentication success/failure rates
- Active sessions count
- Database connection pool metrics

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

This project is part of the SystemSim platform.
