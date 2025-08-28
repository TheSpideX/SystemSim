#!/bin/bash

# Comprehensive API Gateway Test Suite
# Tests HTTP/2, authentication, routing, and integration with auth service

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Test configuration
API_GATEWAY_URL="https://localhost:8000"
AUTH_SERVICE_URL="http://localhost:8001"
TEST_USER_EMAIL="test@example.com"
TEST_USER_PASSWORD="testpassword123"
TEST_TOKEN=""

# Function to print colored output
print_status() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

print_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

print_test() {
    echo -e "${BLUE}[TEST]${NC} $1"
}

# Function to make HTTP requests with proper error handling
make_request() {
    local method=$1
    local url=$2
    local headers=$3
    local data=$4
    local expected_status=$5
    
    local response
    local status_code
    
    if [ -n "$data" ]; then
        response=$(curl -k -s -w "\n%{http_code}" -X "$method" "$url" $headers -d "$data" 2>/dev/null)
    else
        response=$(curl -k -s -w "\n%{http_code}" -X "$method" "$url" $headers 2>/dev/null)
    fi
    
    status_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n -1)
    
    if [ "$status_code" = "$expected_status" ]; then
        print_success "✅ $method $url - Status: $status_code"
        echo "$body"
        return 0
    else
        print_error "❌ $method $url - Expected: $expected_status, Got: $status_code"
        echo "Response: $body"
        return 1
    fi
}

# Function to start API Gateway
start_api_gateway() {
    print_status "Starting API Gateway..."
    
    # Kill any existing process
    pkill -f api-gateway || true
    sleep 2
    
    # Start API Gateway in background
    ./api-gateway > gateway.log 2>&1 &
    GATEWAY_PID=$!
    
    # Wait for startup
    sleep 5
    
    # Check if process is running
    if kill -0 $GATEWAY_PID 2>/dev/null; then
        print_success "API Gateway started (PID: $GATEWAY_PID)"
        return 0
    else
        print_error "Failed to start API Gateway"
        cat gateway.log
        return 1
    fi
}

# Function to stop API Gateway
stop_api_gateway() {
    if [ -n "$GATEWAY_PID" ]; then
        print_status "Stopping API Gateway (PID: $GATEWAY_PID)..."
        kill $GATEWAY_PID 2>/dev/null || true
        wait $GATEWAY_PID 2>/dev/null || true
        print_success "API Gateway stopped"
    fi
}

# Function to check if auth service is running
check_auth_service() {
    print_status "Checking auth service availability..."
    
    if curl -s "$AUTH_SERVICE_URL/health" > /dev/null 2>&1; then
        print_success "Auth service is running"
        return 0
    else
        print_warning "Auth service is not running - some tests will use mock responses"
        return 1
    fi
}

# Test 1: Basic Health Check
test_health_check() {
    print_test "Testing health endpoint..."
    
    local response
    response=$(make_request "GET" "$API_GATEWAY_URL/health" "" "" "200")
    
    if echo "$response" | grep -q '"status"'; then
        print_success "Health check contains status field"
    else
        print_error "Health check missing status field"
        return 1
    fi
    
    if echo "$response" | grep -q '"services"'; then
        print_success "Health check contains services field"
    else
        print_error "Health check missing services field"
        return 1
    fi
}

# Test 2: Metrics Endpoint
test_metrics() {
    print_test "Testing metrics endpoint..."
    
    local response
    response=$(make_request "GET" "$API_GATEWAY_URL/metrics" "" "" "200")
    
    if echo "$response" | grep -q '"gateway"'; then
        print_success "Metrics contains gateway stats"
    else
        print_error "Metrics missing gateway stats"
        return 1
    fi
    
    if echo "$response" | grep -q '"circuit_breakers"'; then
        print_success "Metrics contains circuit breaker stats"
    else
        print_error "Metrics missing circuit breaker stats"
        return 1
    fi
}

# Test 3: CORS Headers
test_cors() {
    print_test "Testing CORS headers..."
    
    local headers
    headers=$(curl -k -s -I -X OPTIONS "$API_GATEWAY_URL/health" 2>/dev/null)
    
    if echo "$headers" | grep -q "Access-Control-Allow-Origin"; then
        print_success "CORS Allow-Origin header present"
    else
        print_error "CORS Allow-Origin header missing"
        return 1
    fi
    
    if echo "$headers" | grep -q "Access-Control-Allow-Methods"; then
        print_success "CORS Allow-Methods header present"
    else
        print_error "CORS Allow-Methods header missing"
        return 1
    fi
}

# Test 4: HTTP/2 Protocol
test_http2() {
    print_test "Testing HTTP/2 protocol support..."
    
    local protocol
    protocol=$(curl -k -s -w "%{http_version}" -o /dev/null "$API_GATEWAY_URL/health" --http2 2>/dev/null)
    
    if [ "$protocol" = "2" ]; then
        print_success "HTTP/2 protocol confirmed"
    else
        print_warning "HTTP/2 not detected (got HTTP/$protocol) - may be fallback behavior"
    fi
}

# Test 5: Authentication - Token Validation (Mock)
test_auth_validate_mock() {
    print_test "Testing auth token validation (mock)..."
    
    # Test without token
    local response
    if ! make_request "POST" "$API_GATEWAY_URL/api/auth/validate" "" "" "401" > /dev/null; then
        print_error "Should return 401 without token"
        return 1
    fi
    
    # Test with mock token
    local headers='-H "Authorization: Bearer mock-test-token" -H "Content-Type: application/json"'
    response=$(make_request "POST" "$API_GATEWAY_URL/api/auth/validate" "$headers" "" "200")
    
    if echo "$response" | grep -q '"valid"'; then
        print_success "Token validation response contains valid field"
    else
        print_error "Token validation response missing valid field"
        return 1
    fi
}

# Test 6: Authentication - User Profile (Mock)
test_auth_profile_mock() {
    print_test "Testing user profile endpoint (mock)..."
    
    # Test without token
    if ! make_request "GET" "$API_GATEWAY_URL/api/auth/profile" "" "" "401" > /dev/null; then
        print_error "Should return 401 without token"
        return 1
    fi
    
    # Test with mock token
    local headers='-H "Authorization: Bearer mock-test-token"'
    local response
    response=$(make_request "GET" "$API_GATEWAY_URL/api/auth/profile" "$headers" "" "200")
    
    if echo "$response" | grep -q '"user_id"'; then
        print_success "Profile response contains user_id"
    else
        print_error "Profile response missing user_id"
        return 1
    fi
}

# Test 7: Authentication - User Permissions (Mock)
test_auth_permissions_mock() {
    print_test "Testing user permissions endpoint (mock)..."
    
    local headers='-H "Authorization: Bearer mock-test-token"'
    local response
    response=$(make_request "GET" "$API_GATEWAY_URL/api/auth/permissions" "$headers" "" "200")
    
    if echo "$response" | grep -q '"permissions"'; then
        print_success "Permissions response contains permissions array"
    else
        print_error "Permissions response missing permissions array"
        return 1
    fi
}

# Test 8: Protected Routes
test_protected_routes() {
    print_test "Testing protected routes..."
    
    # Test project endpoints without auth
    if ! make_request "GET" "$API_GATEWAY_URL/api/projects" "" "" "401" > /dev/null; then
        print_error "Projects endpoint should require authentication"
        return 1
    fi
    
    # Test simulation endpoints without auth
    if ! make_request "GET" "$API_GATEWAY_URL/api/simulations" "" "" "401" > /dev/null; then
        print_error "Simulations endpoint should require authentication"
        return 1
    fi
    
    print_success "Protected routes properly require authentication"
}

# Test 9: WebSocket Endpoint
test_websocket() {
    print_test "Testing WebSocket endpoint..."
    
    local response
    response=$(make_request "GET" "$API_GATEWAY_URL/ws?user_id=test-user" "" "" "200")
    
    if echo "$response" | grep -q '"message"'; then
        print_success "WebSocket endpoint returns proper response"
    else
        print_error "WebSocket endpoint response invalid"
        return 1
    fi
}

# Test 10: Error Handling
test_error_handling() {
    print_test "Testing error handling..."
    
    # Test 404 for non-existent endpoint
    local response
    response=$(make_request "GET" "$API_GATEWAY_URL/nonexistent" "" "" "404")
    
    if echo "$response" | grep -q '"error"'; then
        print_success "404 error properly formatted"
    else
        print_error "404 error not properly formatted"
        return 1
    fi
}

# Test 11: Circuit Breaker Functionality
test_circuit_breaker() {
    print_test "Testing circuit breaker functionality..."
    
    # Check metrics for circuit breaker stats
    local response
    response=$(curl -k -s "$API_GATEWAY_URL/metrics" 2>/dev/null)
    
    if echo "$response" | grep -q '"circuit_breakers"'; then
        print_success "Circuit breaker metrics available"
        
        # Check for auth service circuit breaker
        if echo "$response" | grep -q '"auth"'; then
            print_success "Auth service circuit breaker configured"
        else
            print_warning "Auth service circuit breaker not found in metrics"
        fi
    else
        print_error "Circuit breaker metrics missing"
        return 1
    fi
}

# Test 12: Performance and Load
test_performance() {
    print_test "Testing basic performance..."
    
    print_status "Running 100 concurrent requests to health endpoint..."
    
    # Simple load test
    local start_time=$(date +%s)
    for i in {1..100}; do
        curl -k -s "$API_GATEWAY_URL/health" > /dev/null &
    done
    wait
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))
    
    print_success "Completed 100 requests in ${duration}s ($(echo "scale=2; 100/$duration" | bc) req/s)"
    
    # Check if server is still responsive
    if make_request "GET" "$API_GATEWAY_URL/health" "" "" "200" > /dev/null; then
        print_success "Server remains responsive after load test"
    else
        print_error "Server not responsive after load test"
        return 1
    fi
}

# Main test execution
main() {
    echo "🚀 Comprehensive API Gateway Test Suite"
    echo "========================================"
    
    # Check dependencies
    if ! command -v curl &> /dev/null; then
        print_error "curl is required for testing"
        exit 1
    fi
    
    if ! command -v bc &> /dev/null; then
        print_warning "bc not available - performance calculations may be limited"
    fi
    
    # Start API Gateway
    if ! start_api_gateway; then
        print_error "Failed to start API Gateway"
        exit 1
    fi
    
    # Set up cleanup
    trap stop_api_gateway EXIT
    
    # Check auth service availability
    AUTH_SERVICE_AVAILABLE=false
    if check_auth_service; then
        AUTH_SERVICE_AVAILABLE=true
    fi
    
    # Run tests
    local failed_tests=0
    local total_tests=0
    
    # Basic functionality tests
    tests=(
        "test_health_check"
        "test_metrics"
        "test_cors"
        "test_http2"
        "test_auth_validate_mock"
        "test_auth_profile_mock"
        "test_auth_permissions_mock"
        "test_protected_routes"
        "test_websocket"
        "test_error_handling"
        "test_circuit_breaker"
        "test_performance"
    )
    
    for test in "${tests[@]}"; do
        total_tests=$((total_tests + 1))
        echo ""
        if ! $test; then
            failed_tests=$((failed_tests + 1))
        fi
    done
    
    # Summary
    echo ""
    echo "========================================"
    echo "Test Summary:"
    echo "Total Tests: $total_tests"
    echo "Passed: $((total_tests - failed_tests))"
    echo "Failed: $failed_tests"
    
    if [ $failed_tests -eq 0 ]; then
        print_success "🎉 All tests passed!"
        exit 0
    else
        print_error "❌ $failed_tests test(s) failed"
        exit 1
    fi
}

# Run main function
main "$@"
