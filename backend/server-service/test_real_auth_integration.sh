#!/bin/bash

# Real Auth Service Integration Test
# Tests API Gateway integration with actual auth service via gRPC

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

API_GATEWAY_URL="https://localhost:8000"
AUTH_SERVICE_URL="https://localhost:9001"

print_status() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }
print_test() { echo -e "${BLUE}[TEST]${NC} $1"; }

# Test function
test_endpoint() {
    local method=$1
    local url=$2
    local headers=$3
    local data=$4
    local expected_status=$5
    local test_name=$6
    
    local cmd="curl -k -s -w '\n%{http_code}' -X $method '$url'"
    if [ -n "$headers" ]; then
        cmd="$cmd $headers"
    fi
    if [ -n "$data" ]; then
        cmd="$cmd -d '$data'"
    fi
    
    local response
    response=$(eval $cmd 2>/dev/null)
    local status_code=$(echo "$response" | tail -n1)
    local body=$(echo "$response" | head -n -1)
    
    echo "=== $test_name ==="
    echo "Request: $method $url"
    echo "Expected Status: $expected_status"
    echo "Actual Status: $status_code"
    echo "Response Body: $body"
    echo ""
    
    if [ "$status_code" = "$expected_status" ]; then
        print_success "✅ $test_name"
        return 0
    else
        print_error "❌ $test_name - Expected: $expected_status, Got: $status_code"
        return 1
    fi
}

# Test auth service directly first
test_auth_service_direct() {
    print_test "=== TESTING AUTH SERVICE DIRECTLY ==="
    
    local failed=0
    
    # Test auth service health
    if ! test_endpoint "GET" "$AUTH_SERVICE_URL/health" "" "" "200" "Auth Service Health Check"; then
        failed=$((failed + 1))
    fi
    
    # Test user registration
    local register_data='{"email":"testuser@example.com","password":"ComplexP@ssw0rd2024!","name":"Test User"}'
    if ! test_endpoint "POST" "$AUTH_SERVICE_URL/api/v1/auth/register" "-H 'Content-Type: application/json'" "$register_data" "201" "User Registration"; then
        # User might already exist, try 409
        if ! test_endpoint "POST" "$AUTH_SERVICE_URL/api/v1/auth/register" "-H 'Content-Type: application/json'" "$register_data" "409" "User Registration (Already Exists)"; then
            failed=$((failed + 1))
        fi
    fi
    
    # Test user login
    local login_data='{"email":"testuser@example.com","password":"ComplexP@ssw0rd2024!"}'
    local login_response
    login_response=$(curl -k -s -X POST "$AUTH_SERVICE_URL/api/v1/auth/login" \
        -H "Content-Type: application/json" \
        -d "$login_data" 2>/dev/null)
    
    echo "=== User Login ==="
    echo "Request: POST $AUTH_SERVICE_URL/api/v1/auth/login"
    echo "Response: $login_response"
    echo ""
    
    # Extract token from login response
    local token
    token=$(echo "$login_response" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
    
    if [ -n "$token" ]; then
        print_success "✅ User Login - Token obtained: ${token:0:20}..."
        
        # Test token validation (get user profile)
        if ! test_endpoint "GET" "$AUTH_SERVICE_URL/api/v1/user/profile" "-H 'Authorization: Bearer $token'" "" "200" "Token Validation"; then
            failed=$((failed + 1))
        fi
        
        # Store token for gateway tests
        echo "$token" > /tmp/auth_token.txt
    else
        print_error "❌ User Login - No token received"
        failed=$((failed + 1))
    fi
    
    return $failed
}

# Test API Gateway integration
test_gateway_integration() {
    print_test "=== TESTING API GATEWAY INTEGRATION ==="
    
    local failed=0
    
    # Test gateway health
    if ! test_endpoint "GET" "$API_GATEWAY_URL/health" "" "" "200" "Gateway Health Check"; then
        failed=$((failed + 1))
    fi
    
    # Test auth service health through gateway
    if ! test_endpoint "GET" "$API_GATEWAY_URL/api/auth/health" "" "" "200" "Auth Health via Gateway"; then
        failed=$((failed + 1))
    fi
    
    # Test registration through gateway
    local register_data='{"email":"gatewayuser@example.com","password":"ComplexP@ssw0rd2024!","name":"Gateway User"}'
    if ! test_endpoint "POST" "$API_GATEWAY_URL/api/auth/register" "-H 'Content-Type: application/json'" "$register_data" "201" "Registration via Gateway"; then
        # User might already exist
        if ! test_endpoint "POST" "$API_GATEWAY_URL/api/auth/register" "-H 'Content-Type: application/json'" "$register_data" "409" "Registration via Gateway (Already Exists)"; then
            failed=$((failed + 1))
        fi
    fi
    
    # Test login through gateway
    local login_data='{"email":"gatewayuser@example.com","password":"ComplexP@ssw0rd2024!"}'
    local login_response
    login_response=$(curl -k -s -X POST "$API_GATEWAY_URL/api/auth/login" \
        -H "Content-Type: application/json" \
        -d "$login_data" 2>/dev/null)
    
    echo "=== Login via Gateway ==="
    echo "Request: POST $API_GATEWAY_URL/api/auth/login"
    echo "Response: $login_response"
    echo ""
    
    # Extract token from gateway login
    local gateway_token
    gateway_token=$(echo "$login_response" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
    
    if [ -n "$gateway_token" ]; then
        print_success "✅ Login via Gateway - Token obtained: ${gateway_token:0:20}..."
        
        # Test token validation through gateway
        if ! test_endpoint "POST" "$API_GATEWAY_URL/api/auth/validate" "-H 'Authorization: Bearer $gateway_token'" "" "200" "Token Validation via Gateway"; then
            failed=$((failed + 1))
        fi
        
        # Test user profile through gateway
        if ! test_endpoint "GET" "$API_GATEWAY_URL/api/auth/profile" "-H 'Authorization: Bearer $gateway_token'" "" "200" "User Profile via Gateway"; then
            failed=$((failed + 1))
        fi
        
        # Test user permissions through gateway
        if ! test_endpoint "GET" "$API_GATEWAY_URL/api/auth/permissions" "-H 'Authorization: Bearer $gateway_token'" "" "200" "User Permissions via Gateway"; then
            failed=$((failed + 1))
        fi
        
    else
        print_error "❌ Login via Gateway - No token received"
        failed=$((failed + 1))
    fi
    
    return $failed
}

# Test gRPC connection pooling
test_grpc_connection_pooling() {
    print_test "=== TESTING gRPC CONNECTION POOLING ==="
    
    local failed=0
    
    # Get a token first
    local token
    if [ -f /tmp/auth_token.txt ]; then
        token=$(cat /tmp/auth_token.txt)
    else
        print_error "No token available for connection pooling test"
        return 1
    fi
    
    print_status "Running 20 concurrent requests to test connection pooling..."
    
    # Run multiple concurrent requests to test connection pooling
    local pids=()
    for i in {1..20}; do
        (
            curl -k -s -X POST "$API_GATEWAY_URL/api/auth/validate" \
                -H "Authorization: Bearer $token" > /dev/null 2>&1
            echo $? > /tmp/test_result_$i.txt
        ) &
        pids+=($!)
    done
    
    # Wait for all requests to complete
    for pid in "${pids[@]}"; do
        wait $pid
    done
    
    # Check results
    local success_count=0
    for i in {1..20}; do
        if [ -f /tmp/test_result_$i.txt ]; then
            local result=$(cat /tmp/test_result_$i.txt)
            if [ "$result" = "0" ]; then
                success_count=$((success_count + 1))
            fi
            rm -f /tmp/test_result_$i.txt
        fi
    done
    
    echo "Concurrent Requests: 20"
    echo "Successful Requests: $success_count"
    echo "Success Rate: $(echo "scale=1; $success_count * 100 / 20" | bc 2>/dev/null || echo "N/A")%"
    
    if [ $success_count -ge 18 ]; then  # Allow for some failures
        print_success "✅ gRPC Connection Pooling - $success_count/20 requests successful"
    else
        print_error "❌ gRPC Connection Pooling - Only $success_count/20 requests successful"
        failed=$((failed + 1))
    fi
    
    return $failed
}

# Test protected routes
test_protected_routes() {
    print_test "=== TESTING PROTECTED ROUTES ==="
    
    local failed=0
    
    # Test protected routes without authentication
    if ! test_endpoint "GET" "$API_GATEWAY_URL/api/projects" "" "" "401" "Projects - No Auth"; then
        failed=$((failed + 1))
    fi
    
    if ! test_endpoint "GET" "$API_GATEWAY_URL/api/simulations" "" "" "401" "Simulations - No Auth"; then
        failed=$((failed + 1))
    fi
    
    # Test with authentication (should get service unavailable since project/simulation services aren't running)
    local token
    if [ -f /tmp/auth_token.txt ]; then
        token=$(cat /tmp/auth_token.txt)
        
        if ! test_endpoint "GET" "$API_GATEWAY_URL/api/projects" "-H 'Authorization: Bearer $token'" "" "503" "Projects - With Auth (Service Unavailable)"; then
            failed=$((failed + 1))
        fi
        
        if ! test_endpoint "GET" "$API_GATEWAY_URL/api/simulations" "-H 'Authorization: Bearer $token'" "" "503" "Simulations - With Auth (Service Unavailable)"; then
            failed=$((failed + 1))
        fi
    else
        print_error "No token available for protected route testing"
        failed=$((failed + 1))
    fi
    
    return $failed
}

# Main test execution
main() {
    echo "🔗 Real Auth Service Integration Test Suite"
    echo "============================================"
    
    # Check if services are running
    print_status "Checking service availability..."
    
    if ! curl -k -s "$AUTH_SERVICE_URL/health" > /dev/null 2>&1; then
        print_error "Auth service is not running on $AUTH_SERVICE_URL"
        print_status "Please start the auth service first:"
        print_status "cd ../auth-service && ./main"
        exit 1
    fi
    
    if ! curl -k -s "$API_GATEWAY_URL/health" > /dev/null 2>&1; then
        print_error "API Gateway is not running on $API_GATEWAY_URL"
        print_status "Please start the API Gateway first:"
        print_status "./api-gateway"
        exit 1
    fi
    
    print_success "Both services are running!"
    echo ""
    
    local total_failed=0
    
    # Run test suites
    test_auth_service_direct
    total_failed=$((total_failed + $?))
    
    test_gateway_integration
    total_failed=$((total_failed + $?))
    
    test_grpc_connection_pooling
    total_failed=$((total_failed + $?))
    
    test_protected_routes
    total_failed=$((total_failed + $?))
    
    # Cleanup
    rm -f /tmp/auth_token.txt
    
    # Summary
    echo ""
    echo "============================================"
    echo "🎯 REAL INTEGRATION TEST RESULTS:"
    echo "============================================"
    
    if [ $total_failed -eq 0 ]; then
        print_success "🎉 ALL REAL INTEGRATION TESTS PASSED!"
        echo ""
        echo "✅ Auth Service: Working"
        echo "✅ API Gateway: Working"
        echo "✅ gRPC Integration: Working"
        echo "✅ Connection Pooling: Working"
        echo "✅ Authentication Flow: Working"
        echo "✅ Protected Routes: Working"
        echo ""
        return 0
    else
        print_error "❌ Some integration tests failed"
        return 1
    fi
}

main "$@"
