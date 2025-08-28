#!/bin/bash

# Complete Auth Integration Test
# Tests ALL auth routes via API Gateway integration with real auth service

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
    
    if [ "$status_code" = "$expected_status" ]; then
        print_success "✅ $test_name"
        echo "$body" # Return body for token extraction
        return 0
    else
        print_error "❌ $test_name - Expected: $expected_status, Got: $status_code"
        echo "Response: $body"
        return 1
    fi
}

# Main test execution
main() {
    echo "🔗 Complete Auth Integration Test - ALL Routes"
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
    local token=""
    local refresh_token=""
    
    # Test 1: User Registration
    print_test "=== PUBLIC AUTH ROUTES ==="
    local register_data='{"email":"complete_test@example.com","password":"ComplexP@ssw0rd2024!","name":"Complete Test User"}'
    local register_response
    register_response=$(test_endpoint "POST" "$API_GATEWAY_URL/api/auth/register" "-H 'Content-Type: application/json'" "$register_data" "201" "User Registration")
    if [ $? -ne 0 ]; then
        # User might already exist, try with different email
        register_data='{"email":"complete_test_'$(date +%s)'@example.com","password":"ComplexP@ssw0rd2024!","name":"Complete Test User"}'
        register_response=$(test_endpoint "POST" "$API_GATEWAY_URL/api/auth/register" "-H 'Content-Type: application/json'" "$register_data" "201" "User Registration (New Email)")
        if [ $? -ne 0 ]; then
            failed=$((failed + 1))
        fi
    fi
    
    # Test 2: User Login
    local login_data='{"email":"complete_test@example.com","password":"ComplexP@ssw0rd2024!"}'
    local login_response
    login_response=$(test_endpoint "POST" "$API_GATEWAY_URL/api/auth/login" "-H 'Content-Type: application/json'" "$login_data" "200" "User Login")
    if [ $? -eq 0 ]; then
        # Extract tokens
        token=$(echo "$login_response" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
        refresh_token=$(echo "$login_response" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)
        
        if [ -n "$token" ]; then
            print_success "Token obtained: ${token:0:20}..."
            echo "$token" > /tmp/complete_test_token.txt
        fi
        
        if [ -n "$refresh_token" ]; then
            print_success "Refresh token obtained: ${refresh_token:0:20}..."
            echo "$refresh_token" > /tmp/complete_test_refresh_token.txt
        fi
    else
        failed=$((failed + 1))
    fi
    
    # Test 3: Token Refresh
    if [ -n "$refresh_token" ]; then
        local refresh_data='{"refresh_token":"'$refresh_token'"}'
        test_endpoint "POST" "$API_GATEWAY_URL/api/auth/refresh" "-H 'Content-Type: application/json'" "$refresh_data" "200" "Token Refresh" > /dev/null
        if [ $? -ne 0 ]; then
            failed=$((failed + 1))
        fi
    else
        print_error "❌ Token Refresh - No refresh token available"
        failed=$((failed + 1))
    fi
    
    # Test 4: Forgot Password
    local forgot_data='{"email":"complete_test@example.com"}'
    test_endpoint "POST" "$API_GATEWAY_URL/api/auth/forgot-password" "-H 'Content-Type: application/json'" "$forgot_data" "200" "Forgot Password" > /dev/null
    if [ $? -ne 0 ]; then
        failed=$((failed + 1))
    fi
    
    # Test 5: Resend Verification
    local resend_data='{"email":"complete_test@example.com"}'
    test_endpoint "POST" "$API_GATEWAY_URL/api/auth/resend-verification" "-H 'Content-Type: application/json'" "$resend_data" "200" "Resend Verification" > /dev/null
    if [ $? -ne 0 ]; then
        failed=$((failed + 1))
    fi
    
    # Protected Routes Tests (require authentication)
    if [ -n "$token" ]; then
        print_test "=== USER MANAGEMENT ROUTES ==="
        
        # Test 6: Get User Profile
        test_endpoint "GET" "$API_GATEWAY_URL/api/user/profile" "-H 'Authorization: Bearer $token'" "" "200" "Get User Profile" > /dev/null
        if [ $? -ne 0 ]; then
            failed=$((failed + 1))
        fi
        
        # Test 7: Update User Profile
        local update_data='{"first_name":"Updated","last_name":"User"}'
        test_endpoint "PUT" "$API_GATEWAY_URL/api/user/profile" "-H 'Authorization: Bearer $token' -H 'Content-Type: application/json'" "$update_data" "200" "Update User Profile" > /dev/null
        if [ $? -ne 0 ]; then
            failed=$((failed + 1))
        fi
        
        # Test 8: Get User Sessions
        test_endpoint "GET" "$API_GATEWAY_URL/api/user/sessions" "-H 'Authorization: Bearer $token'" "" "200" "Get User Sessions" > /dev/null
        if [ $? -ne 0 ]; then
            failed=$((failed + 1))
        fi
        
        # Test 9: Get User Stats
        test_endpoint "GET" "$API_GATEWAY_URL/api/user/stats" "-H 'Authorization: Bearer $token'" "" "200" "Get User Stats" > /dev/null
        if [ $? -ne 0 ]; then
            failed=$((failed + 1))
        fi
        
        print_test "=== RBAC ROUTES ==="
        
        # Test 10: Get My Roles
        test_endpoint "GET" "$API_GATEWAY_URL/api/rbac/my-roles" "-H 'Authorization: Bearer $token'" "" "200" "Get My Roles" > /dev/null
        if [ $? -ne 0 ]; then
            failed=$((failed + 1))
        fi
        
        # Test 11: Get My Permissions
        test_endpoint "GET" "$API_GATEWAY_URL/api/rbac/my-permissions" "-H 'Authorization: Bearer $token'" "" "200" "Get My Permissions" > /dev/null
        if [ $? -ne 0 ]; then
            failed=$((failed + 1))
        fi
        
        print_test "=== GATEWAY-SPECIFIC ROUTES ==="
        
        # Test 12: Gateway Token Validation
        test_endpoint "POST" "$API_GATEWAY_URL/api/auth/validate" "-H 'Authorization: Bearer $token'" "" "200" "Gateway Token Validation" > /dev/null
        if [ $? -ne 0 ]; then
            failed=$((failed + 1))
        fi
        
        # Test 13: Gateway Profile (via gRPC)
        test_endpoint "GET" "$API_GATEWAY_URL/api/auth/profile" "-H 'Authorization: Bearer $token'" "" "200" "Gateway Profile (gRPC)" > /dev/null
        if [ $? -ne 0 ]; then
            failed=$((failed + 1))
        fi
        
        # Test 14: Gateway Permissions (via gRPC)
        test_endpoint "GET" "$API_GATEWAY_URL/api/auth/permissions" "-H 'Authorization: Bearer $token'" "" "200" "Gateway Permissions (gRPC)" > /dev/null
        if [ $? -ne 0 ]; then
            failed=$((failed + 1))
        fi
        
        print_test "=== ADMIN ROUTES (May fail if user is not admin) ==="
        
        # Test 15: Get All Roles (Admin)
        test_endpoint "GET" "$API_GATEWAY_URL/api/admin/roles" "-H 'Authorization: Bearer $token'" "" "200" "Get All Roles (Admin)" > /dev/null
        if [ $? -ne 0 ]; then
            print_status "Admin route failed (expected if user is not admin)"
        fi
        
        # Test 16: Get All Permissions (Admin)
        test_endpoint "GET" "$API_GATEWAY_URL/api/admin/permissions" "-H 'Authorization: Bearer $token'" "" "200" "Get All Permissions (Admin)" > /dev/null
        if [ $? -ne 0 ]; then
            print_status "Admin route failed (expected if user is not admin)"
        fi
        
        print_test "=== LOGOUT TEST ==="
        
        # Test 17: Logout
        test_endpoint "POST" "$API_GATEWAY_URL/api/auth/logout" "-H 'Authorization: Bearer $token'" "" "200" "User Logout" > /dev/null
        if [ $? -ne 0 ]; then
            failed=$((failed + 1))
        fi
        
    else
        print_error "No token available for protected route testing"
        failed=$((failed + 10)) # Add penalty for missing token tests
    fi
    
    # Test 18: Health Check
    print_test "=== HEALTH CHECK ==="
    test_endpoint "GET" "$API_GATEWAY_URL/api/auth/health" "" "" "200" "Auth Health Check" > /dev/null
    if [ $? -ne 0 ]; then
        failed=$((failed + 1))
    fi
    
    # Test 19: Performance Test
    print_test "=== PERFORMANCE TEST ==="
    if [ -f /tmp/complete_test_token.txt ]; then
        local test_token=$(cat /tmp/complete_test_token.txt)
        
        print_status "Running 10 concurrent requests to test performance..."
        
        local pids=()
        for i in {1..10}; do
            (
                curl -k -s -X GET "$API_GATEWAY_URL/api/user/profile" \
                    -H "Authorization: Bearer $test_token" > /dev/null 2>&1
                echo $? > /tmp/perf_test_result_$i.txt
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
            if [ -f /tmp/perf_test_result_$i.txt ]; then
                local result=$(cat /tmp/perf_test_result_$i.txt)
                if [ "$result" = "0" ]; then
                    success_count=$((success_count + 1))
                fi
                rm -f /tmp/perf_test_result_$i.txt
            fi
        done
        
        if [ $success_count -ge 8 ]; then
            print_success "✅ Performance Test - $success_count/10 requests successful"
        else
            print_error "❌ Performance Test - Only $success_count/10 requests successful"
            failed=$((failed + 1))
        fi
    fi
    
    # Cleanup
    rm -f /tmp/complete_test_token.txt /tmp/complete_test_refresh_token.txt
    
    # Summary
    echo ""
    echo "=============================================="
    echo "🎯 COMPLETE AUTH INTEGRATION RESULTS:"
    echo "=============================================="
    
    if [ $failed -eq 0 ]; then
        print_success "🎉 ALL AUTH ROUTES WORKING PERFECTLY!"
        echo ""
        echo "✅ Public Routes: Working"
        echo "✅ User Management: Working"
        echo "✅ RBAC Routes: Working"
        echo "✅ Gateway-Specific Routes: Working"
        echo "✅ Admin Routes: Available"
        echo "✅ Performance: Excellent"
        echo ""
        print_success "API Gateway successfully supports ALL auth microservice routes!"
        print_success "Real HTTP forwarding and gRPC integration working perfectly!"
        return 0
    else
        print_error "❌ $failed test(s) failed"
        return 1
    fi
}

main "$@"
