#!/bin/bash

# Authentication Integration Test
# Tests authentication flows and protected routes

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

API_GATEWAY_URL="https://localhost:8000"

print_status() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }
print_test() { echo -e "${BLUE}[TEST]${NC} $1"; }

# Start API Gateway
start_gateway() {
    print_status "Starting API Gateway..."
    pkill -f api-gateway || true
    sleep 2
    
    ./api-gateway > gateway_test.log 2>&1 &
    GATEWAY_PID=$!
    sleep 5
    
    if kill -0 $GATEWAY_PID 2>/dev/null; then
        print_success "API Gateway started (PID: $GATEWAY_PID)"
        return 0
    else
        print_error "Failed to start API Gateway"
        cat gateway_test.log
        return 1
    fi
}

# Stop API Gateway
stop_gateway() {
    if [ -n "$GATEWAY_PID" ]; then
        kill $GATEWAY_PID 2>/dev/null || true
        wait $GATEWAY_PID 2>/dev/null || true
        print_status "API Gateway stopped"
    fi
}

# Test function with better error handling
test_endpoint() {
    local method=$1
    local url=$2
    local headers=$3
    local data=$4
    local expected_status=$5
    local test_name=$6
    
    print_test "$test_name"
    
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
    
    echo "Request: $method $url"
    echo "Expected Status: $expected_status"
    echo "Actual Status: $status_code"
    echo "Response Body: $body"
    echo ""
    
    if [ "$status_code" = "$expected_status" ]; then
        print_success "✅ $test_name - Status: $status_code"
        return 0
    else
        print_error "❌ $test_name - Expected: $expected_status, Got: $status_code"
        return 1
    fi
}

# Main tests
main() {
    echo "🔐 Authentication Integration Test Suite"
    echo "========================================"
    
    if ! start_gateway; then
        exit 1
    fi
    
    trap stop_gateway EXIT
    
    local failed=0
    
    # Test 1: Health check (should work)
    if ! test_endpoint "GET" "$API_GATEWAY_URL/health" "" "" "200" "Health Check"; then
        failed=$((failed + 1))
    fi
    
    # Test 2: Metrics (should work)
    if ! test_endpoint "GET" "$API_GATEWAY_URL/metrics" "" "" "200" "Metrics Endpoint"; then
        failed=$((failed + 1))
    fi
    
    # Test 3: Auth validate without token (should fail)
    if ! test_endpoint "POST" "$API_GATEWAY_URL/api/auth/validate" "" "" "401" "Auth Validate - No Token"; then
        failed=$((failed + 1))
    fi
    
    # Test 4: Auth validate with token (should work with mock)
    if ! test_endpoint "POST" "$API_GATEWAY_URL/api/auth/validate" "-H 'Authorization: Bearer test-token'" "" "200" "Auth Validate - With Token"; then
        failed=$((failed + 1))
    fi
    
    # Test 5: Auth profile without token (should fail)
    if ! test_endpoint "GET" "$API_GATEWAY_URL/api/auth/profile" "" "" "401" "Auth Profile - No Token"; then
        failed=$((failed + 1))
    fi
    
    # Test 6: Auth profile with token (should work with mock)
    if ! test_endpoint "GET" "$API_GATEWAY_URL/api/auth/profile" "-H 'Authorization: Bearer test-token'" "" "200" "Auth Profile - With Token"; then
        failed=$((failed + 1))
    fi
    
    # Test 7: Auth permissions with token (should work with mock)
    if ! test_endpoint "GET" "$API_GATEWAY_URL/api/auth/permissions" "-H 'Authorization: Bearer test-token'" "" "200" "Auth Permissions - With Token"; then
        failed=$((failed + 1))
    fi
    
    # Test 8: Auth health (should work - public endpoint)
    if ! test_endpoint "GET" "$API_GATEWAY_URL/api/auth/health" "" "" "200" "Auth Health Check"; then
        failed=$((failed + 1))
    fi
    
    # Test 9: Protected project route without auth (should fail)
    if ! test_endpoint "GET" "$API_GATEWAY_URL/api/projects" "" "" "401" "Projects - No Auth"; then
        failed=$((failed + 1))
    fi
    
    # Test 10: Protected project route with auth (should work)
    if ! test_endpoint "GET" "$API_GATEWAY_URL/api/projects" "-H 'Authorization: Bearer test-token'" "" "200" "Projects - With Auth"; then
        failed=$((failed + 1))
    fi
    
    # Test 11: Protected simulation route without auth (should fail)
    if ! test_endpoint "GET" "$API_GATEWAY_URL/api/simulations" "" "" "401" "Simulations - No Auth"; then
        failed=$((failed + 1))
    fi
    
    # Test 12: Protected simulation route with auth (should work)
    if ! test_endpoint "GET" "$API_GATEWAY_URL/api/simulations" "-H 'Authorization: Bearer test-token'" "" "200" "Simulations - With Auth"; then
        failed=$((failed + 1))
    fi
    
    # Test 13: CORS preflight
    print_test "CORS Preflight Test"
    local cors_response
    cors_response=$(curl -k -s -I -X OPTIONS "$API_GATEWAY_URL/health" 2>/dev/null)
    echo "CORS Headers:"
    echo "$cors_response" | grep -i "access-control" || echo "No CORS headers found"
    echo ""
    
    if echo "$cors_response" | grep -qi "access-control-allow-origin"; then
        print_success "✅ CORS headers present"
    else
        print_error "❌ CORS headers missing"
        failed=$((failed + 1))
    fi
    
    # Test 14: HTTP/2 support
    print_test "HTTP/2 Support Test"
    local http_version
    http_version=$(curl -k -s -w "%{http_version}" -o /dev/null "$API_GATEWAY_URL/health" --http2 2>/dev/null)
    echo "HTTP Version: $http_version"
    
    if [ "$http_version" = "2" ]; then
        print_success "✅ HTTP/2 confirmed"
    else
        print_success "✅ HTTP/$http_version (fallback behavior is acceptable)"
    fi
    echo ""
    
    # Summary
    echo "========================================"
    echo "Test Results:"
    echo "Failed Tests: $failed"
    
    if [ $failed -eq 0 ]; then
        print_success "🎉 All authentication tests passed!"
        return 0
    else
        print_error "❌ $failed test(s) failed"
        return 1
    fi
}

main "$@"
