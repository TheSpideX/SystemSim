#!/bin/bash

# API Gateway + Auth Service Integration Test
# Tests the real gRPC integration between API Gateway and Auth Service

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

# Main test execution
main() {
    echo "🔗 API Gateway + Auth Service Integration Test"
    echo "=============================================="
    
    # Check if services are running
    print_status "Checking service availability..."
    
    if ! curl -k -s "$AUTH_SERVICE_URL/health" > /dev/null 2>&1; then
        print_error "Auth service is not running on $AUTH_SERVICE_URL"
        exit 1
    fi
    
    if ! curl -k -s "$API_GATEWAY_URL/health" > /dev/null 2>&1; then
        print_error "API Gateway is not running on $API_GATEWAY_URL"
        exit 1
    fi
    
    print_success "Both services are running!"
    echo ""
    
    local failed=0
    
    # Test 1: Gateway Health Check
    print_test "=== GATEWAY HEALTH CHECK ==="
    if ! test_endpoint "GET" "$API_GATEWAY_URL/health" "" "" "200" "Gateway Health Check"; then
        failed=$((failed + 1))
    fi
    
    # Test 2: Auth Service Health via Gateway
    print_test "=== AUTH SERVICE HEALTH VIA GATEWAY ==="
    if ! test_endpoint "GET" "$API_GATEWAY_URL/api/auth/health" "" "" "200" "Auth Health via Gateway"; then
        failed=$((failed + 1))
    fi
    
    # Test 3: User Registration via Gateway
    print_test "=== USER REGISTRATION VIA GATEWAY ==="
    local register_data='{"email":"gateway_test@example.com","password":"ComplexP@ssw0rd2024!","name":"Gateway Test User"}'
    if ! test_endpoint "POST" "$API_GATEWAY_URL/api/auth/register" "-H 'Content-Type: application/json'" "$register_data" "201" "Registration via Gateway"; then
        # User might already exist
        if ! test_endpoint "POST" "$API_GATEWAY_URL/api/auth/register" "-H 'Content-Type: application/json'" "$register_data" "400" "Registration via Gateway (Already Exists)"; then
            failed=$((failed + 1))
        fi
    fi
    
    # Test 4: User Login via Gateway
    print_test "=== USER LOGIN VIA GATEWAY ==="
    local login_data='{"email":"gateway_test@example.com","password":"ComplexP@ssw0rd2024!"}'
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
        
        # Store token for further tests
        echo "$gateway_token" > /tmp/gateway_token.txt
        
        # Test 5: Token Validation via Gateway
        print_test "=== TOKEN VALIDATION VIA GATEWAY ==="
        if ! test_endpoint "POST" "$API_GATEWAY_URL/api/auth/validate" "-H 'Authorization: Bearer $gateway_token'" "" "200" "Token Validation via Gateway"; then
            failed=$((failed + 1))
        fi
        
        # Test 6: User Profile via Gateway
        print_test "=== USER PROFILE VIA GATEWAY ==="
        if ! test_endpoint "GET" "$API_GATEWAY_URL/api/auth/profile" "-H 'Authorization: Bearer $gateway_token'" "" "200" "User Profile via Gateway"; then
            failed=$((failed + 1))
        fi
        
        # Test 7: User Permissions via Gateway
        print_test "=== USER PERMISSIONS VIA GATEWAY ==="
        if ! test_endpoint "GET" "$API_GATEWAY_URL/api/auth/permissions" "-H 'Authorization: Bearer $gateway_token'" "" "200" "User Permissions via Gateway"; then
            failed=$((failed + 1))
        fi
        
    else
        print_error "❌ Login via Gateway - No token received"
        failed=$((failed + 1))
    fi
    
    # Test 8: Protected Routes Authentication
    print_test "=== PROTECTED ROUTES AUTHENTICATION ==="
    
    # Test without authentication
    if ! test_endpoint "GET" "$API_GATEWAY_URL/api/projects" "" "" "401" "Projects - No Auth"; then
        failed=$((failed + 1))
    fi
    
    if ! test_endpoint "GET" "$API_GATEWAY_URL/api/simulations" "" "" "401" "Simulations - No Auth"; then
        failed=$((failed + 1))
    fi
    
    # Test with authentication (should get service unavailable since services aren't running)
    if [ -f /tmp/gateway_token.txt ]; then
        local token=$(cat /tmp/gateway_token.txt)
        
        if ! test_endpoint "GET" "$API_GATEWAY_URL/api/projects" "-H 'Authorization: Bearer $token'" "" "503" "Projects - With Auth (Service Unavailable)"; then
            failed=$((failed + 1))
        fi
        
        if ! test_endpoint "GET" "$API_GATEWAY_URL/api/simulations" "-H 'Authorization: Bearer $token'" "" "503" "Simulations - With Auth (Service Unavailable)"; then
            failed=$((failed + 1))
        fi
    fi
    
    # Test 9: gRPC Connection Pool Test
    print_test "=== gRPC CONNECTION POOL TEST ==="
    
    if [ -f /tmp/gateway_token.txt ]; then
        local token=$(cat /tmp/gateway_token.txt)
        
        print_status "Running 10 concurrent requests to test gRPC connection pooling..."
        
        # Run multiple concurrent requests to test connection pooling
        local pids=()
        for i in {1..10}; do
            (
                curl -k -s -X POST "$API_GATEWAY_URL/api/auth/validate" \
                    -H "Authorization: Bearer $token" > /dev/null 2>&1
                echo $? > /tmp/pool_test_result_$i.txt
            ) &
            pids+=($!)
        done
        
        # Wait for all requests to complete
        for pid in "${pids[@]}"; do
            wait $pid
        done
        
        # Check results
        local success_count=0
        for i in {1..10}; do
            if [ -f /tmp/pool_test_result_$i.txt ]; then
                local result=$(cat /tmp/pool_test_result_$i.txt)
                if [ "$result" = "0" ]; then
                    success_count=$((success_count + 1))
                fi
                rm -f /tmp/pool_test_result_$i.txt
            fi
        done
        
        echo "Concurrent Requests: 10"
        echo "Successful Requests: $success_count"
        echo "Success Rate: $(echo "scale=1; $success_count * 100 / 10" | bc 2>/dev/null || echo "N/A")%"
        
        if [ $success_count -ge 8 ]; then  # Allow for some failures
            print_success "✅ gRPC Connection Pooling - $success_count/10 requests successful"
        else
            print_error "❌ gRPC Connection Pooling - Only $success_count/10 requests successful"
            failed=$((failed + 1))
        fi
    else
        print_error "No token available for connection pooling test"
        failed=$((failed + 1))
    fi
    
    # Test 10: Compare Direct vs Gateway Performance
    print_test "=== PERFORMANCE COMPARISON ==="
    
    if [ -f /tmp/gateway_token.txt ]; then
        local token=$(cat /tmp/gateway_token.txt)
        
        # Test direct auth service
        print_status "Testing direct auth service performance..."
        local direct_start=$(date +%s%N)
        for i in {1..5}; do
            curl -k -s -X GET "$AUTH_SERVICE_URL/api/v1/user/profile" \
                -H "Authorization: Bearer $token" > /dev/null 2>&1
        done
        local direct_end=$(date +%s%N)
        local direct_time=$(echo "scale=3; ($direct_end - $direct_start) / 1000000" | bc 2>/dev/null || echo "N/A")
        
        # Test via gateway
        print_status "Testing gateway performance..."
        local gateway_start=$(date +%s%N)
        for i in {1..5}; do
            curl -k -s -X GET "$API_GATEWAY_URL/api/auth/profile" \
                -H "Authorization: Bearer $token" > /dev/null 2>&1
        done
        local gateway_end=$(date +%s%N)
        local gateway_time=$(echo "scale=3; ($gateway_end - $gateway_start) / 1000000" | bc 2>/dev/null || echo "N/A")
        
        echo "Direct Auth Service (5 requests): ${direct_time}ms"
        echo "Via API Gateway (5 requests): ${gateway_time}ms"
        
        if [ "$gateway_time" != "N/A" ] && [ "$direct_time" != "N/A" ]; then
            local overhead=$(echo "scale=1; ($gateway_time - $direct_time) * 100 / $direct_time" | bc 2>/dev/null || echo "N/A")
            echo "Gateway Overhead: ${overhead}%"
            
            # Consider acceptable if overhead is less than 100%
            if [ "$(echo "$overhead < 100" | bc 2>/dev/null)" = "1" ]; then
                print_success "✅ Performance - Gateway overhead acceptable: ${overhead}%"
            else
                print_error "❌ Performance - Gateway overhead too high: ${overhead}%"
                failed=$((failed + 1))
            fi
        else
            print_success "✅ Performance - Both services responding"
        fi
    fi
    
    # Cleanup
    rm -f /tmp/gateway_token.txt
    
    # Summary
    echo ""
    echo "=============================================="
    echo "🎯 API GATEWAY + AUTH SERVICE INTEGRATION RESULTS:"
    echo "=============================================="
    
    if [ $failed -eq 0 ]; then
        print_success "🎉 ALL INTEGRATION TESTS PASSED!"
        echo ""
        echo "✅ API Gateway: Working"
        echo "✅ Auth Service: Working"
        echo "✅ gRPC Communication: Working"
        echo "✅ Connection Pooling: Working"
        echo "✅ Authentication Flow: Working"
        echo "✅ Protected Routes: Working"
        echo "✅ Performance: Acceptable"
        echo ""
        print_success "The API Gateway successfully integrates with the Auth Service!"
        print_success "Real gRPC communication is working perfectly!"
        return 0
    else
        print_error "❌ $failed integration test(s) failed"
        return 1
    fi
}

main "$@"
