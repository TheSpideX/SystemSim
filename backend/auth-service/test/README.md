# Auth Service Testing Guide

This directory contains comprehensive tests for the auth microservice that test the **actual running service** rather than mocks or containers.

## Test Types

### 1. **Functional Tests** (`functional_test.go`)
Tests the HTTP API endpoints of the running auth service:
- Complete user registration flow with validation
- Login/logout functionality with session management
- Password security validation (weak passwords rejected)
- Account lockout after 5 failed attempts
- Input validation and security (XSS, SQL injection prevention)
- Token security and JWT validation
- Session management with remember me functionality

### 2. **gRPC Tests** (`grpc_test.go`)
Tests the gRPC endpoints and connection behavior:
- gRPC connection establishment and health
- Concurrent connection handling
- Connection pooling behavior (5-20 connections)
- Connection resilience and error handling
- Performance baseline testing

### 3. **Load Tests** (`load_test.go`)
Tests system performance under concurrent load:
- Concurrent user registration (50 users, 10 concurrent)
- Concurrent login testing (20 users × 3 logins, 15 concurrent)
- Mixed operations stress test (100 operations, 20 concurrent)
- Performance metrics and throughput validation

### 4. **Security Tests** (`../internal/security/`)
Unit tests for security components:
- Password validation and bcrypt hashing
- JWT token generation, validation, and expiry
- Security functionality without external dependencies

## Prerequisites

Before running tests, you need:

1. **PostgreSQL running** on localhost:5432 with database `auth_service_dev`
2. **Redis running** on localhost:6379
3. **Auth service running** on:
   - HTTP/2: localhost:9001 (with TLS)
   - gRPC: localhost:9000

## Running Tests

### Quick Test (Unit Tests Only)
```bash
# Run security unit tests (no external dependencies)
go test -v ./internal/security/

# Run with short mode (skips load tests)
go test -v ./test/ -short
```

### Full Functional Testing
```bash
# 1. Start the auth service first
go run cmd/main.go

# 2. In another terminal, run functional tests
go test -v ./test/ -run TestAuthServiceFunctionality

# 3. Run gRPC tests
go test -v ./test/ -run TestAuthServiceGRPC

# 4. Run load tests (not in short mode)
go test -v ./test/ -run TestAuthServiceLoad
```

### All Tests
```bash
# Run all tests (requires running service)
go test -v ./test/

# Run all tests including security unit tests
go test -v ./internal/security/ ./test/
```

## Test Behavior

### **Automatic Skipping**
Tests automatically skip if dependencies aren't available:
- HTTP/2 tests skip if service not running on localhost:9001
- gRPC tests skip if service not running on localhost:9000
- Load tests skip in `-short` mode

### **Real Functionality Testing**
These tests validate **actual system behavior**:
- ✅ **Registration** creates real users in database
- ✅ **Login** validates actual password hashing
- ✅ **Account lockout** tests real rate limiting (5 attempts → 15min lock)
- ✅ **Security validation** tests real input sanitization
- ✅ **JWT tokens** are real tokens with proper expiry
- ✅ **Session management** tests real Redis sessions
- ✅ **Load testing** measures real performance metrics

### **Test Data Cleanup**
Tests use unique identifiers to avoid conflicts:
- Email addresses include timestamps: `test_1642123456@example.com`
- Multiple test runs won't interfere with each other
- No manual cleanup required between test runs

## Expected Results

### **Performance Benchmarks**
- **Registration**: ≥5 registrations/second under load
- **Login**: ≥10 logins/second under load
- **Mixed operations**: ≥5 operations/second under stress
- **gRPC**: ≥1000 operations/second baseline

### **Security Validation**
- Weak passwords properly rejected
- Account lockout after 5 failed attempts
- Malicious inputs (XSS, SQL injection) blocked
- JWT tokens properly validated and expired
- Session security with IP/User-Agent binding

### **Reliability**
- ≥90% success rate under concurrent load
- ≥80% success rate under stress conditions
- Graceful error handling and proper HTTP status codes
- Connection pooling scales from 5-20 connections

## Example Test Output

```bash
=== RUN   TestAuthServiceFunctionality/complete_user_registration_flow
    functional_test.go:45: Registration successful for test_1642123456@example.com
    functional_test.go:67: Login successful with new tokens
    functional_test.go:89: Profile access successful
    functional_test.go:101: Token refresh successful
    functional_test.go:113: Logout successful
    functional_test.go:118: Old token properly invalidated
--- PASS: TestAuthServiceFunctionality/complete_user_registration_flow (0.15s)

=== RUN   TestAuthServiceLoad/concurrent_user_registration_load
    load_test.go:78: Concurrent registration load test results:
    load_test.go:79: - Total users: 50
    load_test.go:80: - Concurrency: 10
    load_test.go:81: - Successful: 49
    load_test.go:82: - Failed: 1
    load_test.go:83: - Duration: 8.2s
    load_test.go:84: - Throughput: 5.98 registrations/second
--- PASS: TestAuthServiceLoad/concurrent_user_registration_load (8.20s)
```

## Why This Approach?

### **Advantages over Container/Mock Tests:**
1. **Faster execution** - no container startup time
2. **Real environment testing** - tests actual deployed service
3. **Simple setup** - just HTTP/gRPC calls
4. **Easy debugging** - can see service logs directly
5. **Production-like** - tests the actual API interface
6. **No Docker required** - works on any system

### **Deep Functionality Testing:**
- Tests validate **actual business logic**, not just "passing tests"
- Security tests find **real vulnerabilities** in input validation
- Load tests measure **real performance** under concurrent users
- Error scenarios test **actual failure handling**
- Integration tests validate **real database/Redis operations**

This testing approach ensures the auth microservice works correctly in real-world conditions with actual users, concurrent load, and security threats.
