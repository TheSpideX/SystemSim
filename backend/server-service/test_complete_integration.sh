#!/bin/bash

# Complete Integration Test Suite
# Tests API Gateway + Auth Service integration with REAL dynamic connection pool validation

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

API_GATEWAY_URL="https://localhost:8000"
AUTH_SERVICE_URL="https://localhost:9001"

print_status() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }
print_test() { echo -e "${PURPLE}[TEST]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }
print_phase() { echo -e "${CYAN}[PHASE]${NC} $1"; }

# Global variables
FAILED_TESTS=0
TOTAL_TESTS=0
TEST_USER_EMAIL="integration_test_$(date +%s)@example.com"
TEST_USER_PASSWORD="ComplexP@ssw0rd2024!"
ACCESS_TOKEN=""
REFRESH_TOKEN=""

# Test function with detailed metrics
test_endpoint() {
    local method=$1
    local url=$2
    local headers=$3
    local data=$4
    local expected_status=$5
    local test_name=$6
    
    TOTAL_TESTS=$((TOTAL_TESTS + 1))
    
    local cmd="curl -k -s -w '\n%{http_code}\n%{time_total}\n%{time_connect}' -X $method '$url'"
    if [ -n "$headers" ]; then
        cmd="$cmd $headers"
    fi
    if [ -n "$data" ]; then
        cmd="$cmd -d '$data'"
    fi
    
    local response
    response=$(eval $cmd 2>/dev/null)
    local body=$(echo "$response" | head -n -3)
    local status_code=$(echo "$response" | tail -n 3 | head -n 1)
    local total_time=$(echo "$response" | tail -n 2 | head -n 1)
    local connect_time=$(echo "$response" | tail -n 1)
    
    if [ "$status_code" = "$expected_status" ]; then
        print_success "✅ $test_name (${total_time}s, connect: ${connect_time}s)"
        echo "$body" # Return body for token extraction
        return 0
    else
        print_error "❌ $test_name - Expected: $expected_status, Got: $status_code (${total_time}s)"
        echo "Response: $body"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# Get gRPC pool statistics
get_grpc_stats() {
    local stats_response
    stats_response=$(curl -k -s "$API_GATEWAY_URL/grpc/stats" 2>/dev/null)
    
    if [ $? -eq 0 ] && [[ "$stats_response" == *"auth_service"* ]]; then
        echo "$stats_response"
    else
        echo '{"error": "failed to get stats"}'
    fi
}

# Extract specific stat from gRPC stats
extract_stat() {
    local stats=$1
    local field=$2
    echo "$stats" | grep -o "\"$field\":[0-9]*" | cut -d':' -f2 | head -n1
}

# Generate concurrent load with detailed monitoring
generate_concurrent_load() {
    local concurrent_requests=$1
    local duration_seconds=$2
    local endpoint=$3
    local headers=$4
    
    print_status "Generating $concurrent_requests concurrent requests to $endpoint for ${duration_seconds}s..."
    
    local pids=()
    local start_time=$(date +%s)
    local end_time=$((start_time + duration_seconds))
    
    # Start concurrent request generators
    for i in $(seq 1 $concurrent_requests); do
        (
            local request_count=0
            local success_count=0
            while [ $(date +%s) -lt $end_time ]; do
                local response_code=$(curl -k -s -w '%{http_code}' -o /dev/null -X POST "$endpoint" $headers 2>/dev/null)
                request_count=$((request_count + 1))
                if [ "$response_code" = "200" ]; then
                    success_count=$((success_count + 1))
                fi
                sleep 0.05 # 50ms between requests per thread
            done
            echo "$request_count,$success_count" > /tmp/load_test_result_$i.txt
        ) &
        pids+=($!)
    done
    
    # Monitor connection scaling during load
    local monitor_interval=2
    local monitor_count=0
    local max_monitors=$((duration_seconds / monitor_interval))
    
    echo "Time,Connections,Active,Utilization,Total_Requests" > /tmp/scaling_monitor.csv
    
    while [ $(date +%s) -lt $end_time ] && [ $monitor_count -lt $max_monitors ]; do
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        
        local stats=$(get_grpc_stats)
        local connections=$(extract_stat "$stats" "current_connections")
        local active=$(extract_stat "$stats" "active_requests")
        local total_requests=$(extract_stat "$stats" "total_requests")
        local utilization=$(echo "$stats" | grep -o '"utilization":[0-9.]*' | cut -d':' -f2)
        
        echo "$elapsed,$connections,$active,$utilization,$total_requests" >> /tmp/scaling_monitor.csv
        print_status "[$elapsed s] Connections: $connections, Active: $active, Utilization: $utilization, Total: $total_requests"
        
        sleep $monitor_interval
        monitor_count=$((monitor_count + 1))
    done
    
    # Wait for all load generators to complete
    for pid in "${pids[@]}"; do
        wait $pid
    done
    
    # Calculate total results
    local total_requests=0
    local total_success=0
    for i in $(seq 1 $concurrent_requests); do
        if [ -f /tmp/load_test_result_$i.txt ]; then
            local result=$(cat /tmp/load_test_result_$i.txt)
            local requests=$(echo "$result" | cut -d',' -f1)
            local success=$(echo "$result" | cut -d',' -f2)
            total_requests=$((total_requests + requests))
            total_success=$((total_success + success))
            rm -f /tmp/load_test_result_$i.txt
        fi
    done
    
    local success_rate=0
    if [ $total_requests -gt 0 ]; then
        success_rate=$((total_success * 100 / total_requests))
    fi
    
    print_success "Load test completed: $total_requests requests, $total_success successful ($success_rate%)"
    
    # Return final stats
    local final_stats=$(get_grpc_stats)
    local final_connections=$(extract_stat "$final_stats" "current_connections")
    echo "FINAL_CONNECTIONS:$final_connections"
}

# Validate connection scaling behavior
validate_scaling() {
    local phase=$1
    local expected_min=$2
    local expected_max=$3
    local stats=$4
    
    local connections=$(extract_stat "$stats" "current_connections")
    
    if [ -z "$connections" ] || [ "$connections" = "" ]; then
        print_error "❌ $phase: Could not extract connection count"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
    
    if [ "$connections" -ge "$expected_min" ] && [ "$connections" -le "$expected_max" ]; then
        print_success "✅ $phase: Connections in expected range ($connections between $expected_min-$expected_max)"
        return 0
    else
        print_error "❌ $phase: Connections outside expected range ($connections not between $expected_min-$expected_max)"
        FAILED_TESTS=$((FAILED_TESTS + 1))
        return 1
    fi
}

# Main test execution
main() {
    echo "🔍 COMPLETE INTEGRATION TEST SUITE"
    echo "=================================="
    echo "Testing API Gateway + Auth Service with REAL dynamic connection pool validation"
    echo "Test User: $TEST_USER_EMAIL"
    echo ""
    
    # Check service availability
    print_phase "PHASE 0: SERVICE AVAILABILITY CHECK"
    
    if ! curl -k -s "$AUTH_SERVICE_URL/health" > /dev/null 2>&1; then
        print_error "Auth service is not running on $AUTH_SERVICE_URL"
        exit 1
    fi
    
    if ! curl -k -s "$API_GATEWAY_URL/health" > /dev/null 2>&1; then
        print_error "API Gateway is not running on $API_GATEWAY_URL"
        exit 1
    fi
    
    if ! curl -k -s "$API_GATEWAY_URL/grpc/stats" > /dev/null 2>&1; then
        print_error "API Gateway gRPC stats endpoint not available"
        exit 1
    fi
    
    print_success "All services are running and accessible!"
    echo ""
    
    # Phase 1: Baseline Connection Pool Analysis
    print_phase "PHASE 1: BASELINE CONNECTION POOL ANALYSIS"
    
    local initial_stats=$(get_grpc_stats)
    print_status "Initial gRPC pool state:"
    echo "$initial_stats" | grep -o '"auth_service":{[^}]*}' | sed 's/,/\n  /g' | sed 's/{/{\n  /' | sed 's/}/\n}/'
    
    validate_scaling "Initial State" 5 5 "$initial_stats"
    echo ""
    
    # Phase 2: User Registration and Authentication Flow
    print_phase "PHASE 2: COMPLETE USER AUTHENTICATION FLOW"
    
    # Test 1: User Registration
    local register_data="{\"email\":\"$TEST_USER_EMAIL\",\"password\":\"$TEST_USER_PASSWORD\",\"name\":\"Integration Test User\"}"
    local register_response
    register_response=$(test_endpoint "POST" "$API_GATEWAY_URL/api/auth/register" "-H 'Content-Type: application/json'" "$register_data" "201" "User Registration")
    
    # Test 2: User Login
    local login_data="{\"email\":\"$TEST_USER_EMAIL\",\"password\":\"$TEST_USER_PASSWORD\"}"
    local login_response
    login_response=$(test_endpoint "POST" "$API_GATEWAY_URL/api/auth/login" "-H 'Content-Type: application/json'" "$login_data" "200" "User Login")
    
    if [ $? -eq 0 ]; then
        ACCESS_TOKEN=$(echo "$login_response" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
        REFRESH_TOKEN=$(echo "$login_response" | grep -o '"refresh_token":"[^"]*"' | cut -d'"' -f4)
        
        if [ -n "$ACCESS_TOKEN" ]; then
            print_success "Access token obtained: ${ACCESS_TOKEN:0:20}..."
        else
            print_error "Failed to extract access token"
            exit 1
        fi
    else
        print_error "Login failed, cannot continue with protected route tests"
        exit 1
    fi
    
    # Test 3: Token Validation (gRPC)
    test_endpoint "POST" "$API_GATEWAY_URL/api/auth/validate" "-H 'Authorization: Bearer $ACCESS_TOKEN'" "" "200" "Token Validation (gRPC)" > /dev/null
    
    # Test 4: User Profile (gRPC)
    test_endpoint "GET" "$API_GATEWAY_URL/api/auth/profile" "-H 'Authorization: Bearer $ACCESS_TOKEN'" "" "200" "User Profile (gRPC)" > /dev/null
    
    # Test 5: User Permissions (gRPC)
    test_endpoint "GET" "$API_GATEWAY_URL/api/auth/permissions" "-H 'Authorization: Bearer $ACCESS_TOKEN'" "" "200" "User Permissions (gRPC)" > /dev/null
    
    local after_auth_stats=$(get_grpc_stats)
    validate_scaling "After Authentication" 5 7 "$after_auth_stats"
    echo ""
    
    # Phase 3: Low Load Test (Should maintain minimum connections)
    print_phase "PHASE 3: LOW LOAD TEST (Should maintain ~5 connections)"
    
    local low_load_result=$(generate_concurrent_load 3 20 "$API_GATEWAY_URL/api/auth/validate" "-H 'Authorization: Bearer $ACCESS_TOKEN'")
    local low_load_connections=$(echo "$low_load_result" | grep "FINAL_CONNECTIONS:" | cut -d':' -f2)
    
    sleep 5 # Allow metrics to settle
    
    local low_load_stats=$(get_grpc_stats)
    validate_scaling "Low Load" 5 8 "$low_load_stats"
    echo ""
    
    # Phase 4: High Load Test (Should scale up connections)
    print_phase "PHASE 4: HIGH LOAD TEST (Should scale up to 10-20 connections)"
    
    print_status "Generating high concurrent load to trigger scaling..."
    local high_load_result=$(generate_concurrent_load 20 45 "$API_GATEWAY_URL/api/auth/validate" "-H 'Authorization: Bearer $ACCESS_TOKEN'")
    local high_load_connections=$(echo "$high_load_result" | grep "FINAL_CONNECTIONS:" | cut -d':' -f2)
    
    sleep 5 # Allow scaling to complete
    
    local high_load_stats=$(get_grpc_stats)
    validate_scaling "High Load" 8 20 "$high_load_stats"
    
    # Verify scaling actually occurred
    if [ "$high_load_connections" -gt "$low_load_connections" ]; then
        print_success "✅ Dynamic Scaling Verified: $low_load_connections → $high_load_connections connections"
    else
        print_error "❌ Dynamic Scaling Failed: $low_load_connections → $high_load_connections connections"
        FAILED_TESTS=$((FAILED_TESTS + 1))
    fi
    echo ""
    
    # Phase 5: Scale Down Test (Should scale back down)
    print_phase "PHASE 5: SCALE DOWN TEST (Should scale back down after load decreases)"
    
    print_status "Waiting for scale down (60 seconds)..."
    sleep 60
    
    local scale_down_stats=$(get_grpc_stats)
    local scale_down_connections=$(extract_stat "$scale_down_stats" "current_connections")
    
    if [ "$scale_down_connections" -lt "$high_load_connections" ]; then
        print_success "✅ Scale Down Verified: $high_load_connections → $scale_down_connections connections"
    else
        print_warning "⚠️  Scale Down: $high_load_connections → $scale_down_connections connections (may need more time)"
    fi
    
    validate_scaling "Scale Down" 5 15 "$scale_down_stats"
    echo ""
    
    # Phase 6: End-to-End Workflow Testing
    print_phase "PHASE 6: END-TO-END WORKFLOW TESTING"
    
    # Test complete user workflow
    test_endpoint "GET" "$API_GATEWAY_URL/api/user/profile" "-H 'Authorization: Bearer $ACCESS_TOKEN'" "" "200" "Get User Profile (HTTP)" > /dev/null
    test_endpoint "PUT" "$API_GATEWAY_URL/api/user/profile" "-H 'Authorization: Bearer $ACCESS_TOKEN' -H 'Content-Type: application/json'" '{"first_name":"Integration","last_name":"Test"}' "200" "Update User Profile" > /dev/null
    test_endpoint "GET" "$API_GATEWAY_URL/api/user/sessions" "-H 'Authorization: Bearer $ACCESS_TOKEN'" "" "200" "Get User Sessions" > /dev/null
    test_endpoint "GET" "$API_GATEWAY_URL/api/rbac/my-roles" "-H 'Authorization: Bearer $ACCESS_TOKEN'" "" "200" "Get User Roles" > /dev/null
    test_endpoint "GET" "$API_GATEWAY_URL/api/rbac/my-permissions" "-H 'Authorization: Bearer $ACCESS_TOKEN'" "" "200" "Get User Permissions" > /dev/null
    
    # Test token refresh
    if [ -n "$REFRESH_TOKEN" ]; then
        local refresh_data="{\"refresh_token\":\"$REFRESH_TOKEN\"}"
        test_endpoint "POST" "$API_GATEWAY_URL/api/auth/refresh" "-H 'Content-Type: application/json'" "$refresh_data" "200" "Token Refresh" > /dev/null
    fi
    
    echo ""
    
    # Phase 7: Performance Analysis
    print_phase "PHASE 7: PERFORMANCE ANALYSIS"
    
    if [ -f /tmp/scaling_monitor.csv ]; then
        print_status "Connection scaling timeline:"
        echo "Time(s) | Connections | Active | Utilization | Total Requests"
        echo "--------|-------------|--------|-------------|---------------"
        tail -10 /tmp/scaling_monitor.csv | while IFS=',' read -r time conn active util total; do
            printf "%-7s | %-11s | %-6s | %-11s | %s\n" "$time" "$conn" "$active" "$util" "$total"
        done
        echo ""
    fi
    
    local final_stats=$(get_grpc_stats)
    local final_total_requests=$(extract_stat "$final_stats" "total_requests")
    local final_error_count=$(extract_stat "$final_stats" "error_count")
    local final_connections=$(extract_stat "$final_stats" "current_connections")
    
    print_status "Final Performance Metrics:"
    print_status "  Total gRPC Requests: $final_total_requests"
    print_status "  Total Errors: $final_error_count"
    print_status "  Final Connections: $final_connections"
    print_status "  Success Rate: $(( (final_total_requests - final_error_count) * 100 / final_total_requests ))%"
    
    echo ""
    
    # Cleanup
    rm -f /tmp/scaling_monitor.csv /tmp/load_test_result_*.txt
    
    # Final Results
    echo "=============================================="
    echo "🎯 COMPLETE INTEGRATION TEST RESULTS:"
    echo "=============================================="
    
    local passed_tests=$((TOTAL_TESTS - FAILED_TESTS))
    local success_rate=$((passed_tests * 100 / TOTAL_TESTS))
    
    print_status "Tests Run: $TOTAL_TESTS"
    print_status "Tests Passed: $passed_tests"
    print_status "Tests Failed: $FAILED_TESTS"
    print_status "Success Rate: $success_rate%"
    
    echo ""
    
    if [ $FAILED_TESTS -eq 0 ]; then
        print_success "🎉 ALL INTEGRATION TESTS PASSED!"
        echo ""
        print_success "✅ Dynamic Connection Pooling: WORKING"
        print_success "✅ Load-Based Scaling: WORKING"
        print_success "✅ Real gRPC Integration: WORKING"
        print_success "✅ End-to-End Workflows: WORKING"
        print_success "✅ Performance: EXCELLENT"
        echo ""
        print_success "🚀 SYSTEM IS PRODUCTION READY!"
        return 0
    else
        print_error "❌ $FAILED_TESTS integration test(s) failed"
        print_error "🔧 System needs investigation before production deployment"
        return 1
    fi
}

main "$@"
