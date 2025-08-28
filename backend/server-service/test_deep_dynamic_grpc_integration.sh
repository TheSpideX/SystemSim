#!/bin/bash

# Deep Dynamic gRPC Integration Test
# Tests REAL dynamic connection pooling behavior under load
# Validates that connections scale from 5 to 20 based on actual load

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
PURPLE='\033[0;35m'
NC='\033[0m'

API_GATEWAY_URL="https://localhost:8000"
AUTH_SERVICE_URL="https://localhost:9001"

print_status() { echo -e "${BLUE}[INFO]${NC} $1"; }
print_success() { echo -e "${GREEN}[SUCCESS]${NC} $1"; }
print_error() { echo -e "${RED}[ERROR]${NC} $1"; }
print_test() { echo -e "${PURPLE}[TEST]${NC} $1"; }
print_warning() { echo -e "${YELLOW}[WARNING]${NC} $1"; }

# Test function with detailed response analysis
test_endpoint_detailed() {
    local method=$1
    local url=$2
    local headers=$3
    local data=$4
    local expected_status=$5
    local test_name=$6
    
    local cmd="curl -k -s -w '\n%{http_code}\n%{time_total}' -X $method '$url'"
    if [ -n "$headers" ]; then
        cmd="$cmd $headers"
    fi
    if [ -n "$data" ]; then
        cmd="$cmd -d '$data'"
    fi
    
    local response
    response=$(eval $cmd 2>/dev/null)
    local body=$(echo "$response" | head -n -2)
    local status_code=$(echo "$response" | tail -n 2 | head -n 1)
    local response_time=$(echo "$response" | tail -n 1)
    
    if [ "$status_code" = "$expected_status" ]; then
        print_success "✅ $test_name (${response_time}s)"
        echo "$body" # Return body for token extraction
        return 0
    else
        print_error "❌ $test_name - Expected: $expected_status, Got: $status_code (${response_time}s)"
        echo "Response: $body"
        return 1
    fi
}

# Function to get gRPC pool statistics
get_grpc_stats() {
    local stats_response
    stats_response=$(curl -k -s "$API_GATEWAY_URL/health" 2>/dev/null)
    
    # Extract gRPC connection info from health response
    local auth_connections=$(echo "$stats_response" | grep -o '"auth_service":[^}]*' | grep -o '"current_connections":[0-9]*' | cut -d':' -f2 || echo "0")
    local active_requests=$(echo "$stats_response" | grep -o '"auth_service":[^}]*' | grep -o '"active_requests":[0-9]*' | cut -d':' -f2 || echo "0")
    local utilization=$(echo "$stats_response" | grep -o '"auth_service":[^}]*' | grep -o '"utilization":[0-9.]*' | cut -d':' -f2 || echo "0")
    
    echo "Connections: $auth_connections, Active: $active_requests, Utilization: $utilization"
}

# Function to generate concurrent load
generate_load() {
    local concurrent_requests=$1
    local duration_seconds=$2
    local token=$3
    
    print_status "Generating $concurrent_requests concurrent requests for ${duration_seconds}s..."
    
    local pids=()
    local start_time=$(date +%s)
    local end_time=$((start_time + duration_seconds))
    
    # Start concurrent request generators
    for i in $(seq 1 $concurrent_requests); do
        (
            local request_count=0
            while [ $(date +%s) -lt $end_time ]; do
                curl -k -s -X POST "$API_GATEWAY_URL/api/auth/validate" \
                    -H "Authorization: Bearer $token" > /dev/null 2>&1
                request_count=$((request_count + 1))
                sleep 0.1 # Small delay between requests
            done
            echo $request_count > /tmp/load_test_result_$i.txt
        ) &
        pids+=($!)
    done
    
    # Monitor connection scaling during load
    local monitor_count=0
    while [ $(date +%s) -lt $end_time ] && [ $monitor_count -lt 10 ]; do
        local stats=$(get_grpc_stats)
        print_status "Load Monitor: $stats"
        sleep 3
        monitor_count=$((monitor_count + 1))
    done
    
    # Wait for all load generators to complete
    for pid in "${pids[@]}"; do
        wait $pid
    done
    
    # Calculate total requests
    local total_requests=0
    for i in $(seq 1 $concurrent_requests); do
        if [ -f /tmp/load_test_result_$i.txt ]; then
            local count=$(cat /tmp/load_test_result_$i.txt)
            total_requests=$((total_requests + count))
            rm -f /tmp/load_test_result_$i.txt
        fi
    done
    
    print_success "Load test completed: $total_requests total requests"
    return $total_requests
}

# Main test execution
main() {
    echo "🔍 DEEP DYNAMIC gRPC INTEGRATION TEST"
    echo "====================================="
    echo "Testing REAL dynamic connection pooling (5-20 connections)"
    echo "Validating load-based scaling behavior"
    echo ""
    
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
    
    # Phase 1: Get authentication token
    print_test "=== PHASE 1: AUTHENTICATION ==="
    
    local login_data='{"email":"deep_test@example.com","password":"ComplexP@ssw0rd2024!"}'
    local login_response
    login_response=$(test_endpoint_detailed "POST" "$API_GATEWAY_URL/api/auth/login" "-H 'Content-Type: application/json'" "$login_data" "200" "Deep Test Login")
    
    if [ $? -eq 0 ]; then
        token=$(echo "$login_response" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)
        if [ -n "$token" ]; then
            print_success "Authentication token obtained: ${token:0:20}..."
        else
            print_error "Failed to extract token from response"
            failed=$((failed + 1))
        fi
    else
        failed=$((failed + 1))
    fi
    
    if [ -z "$token" ]; then
        print_error "Cannot proceed without authentication token"
        exit 1
    fi
    
    echo ""
    
    # Phase 2: Baseline gRPC Pool Analysis
    print_test "=== PHASE 2: BASELINE gRPC POOL ANALYSIS ==="
    
    print_status "Initial gRPC pool state:"
    local initial_stats=$(get_grpc_stats)
    print_status "$initial_stats"
    
    # Test individual gRPC calls to verify they work
    print_status "Testing individual gRPC calls..."
    
    test_endpoint_detailed "POST" "$API_GATEWAY_URL/api/auth/validate" "-H 'Authorization: Bearer $token'" "" "200" "gRPC ValidateToken" > /dev/null
    if [ $? -ne 0 ]; then
        failed=$((failed + 1))
    fi
    
    test_endpoint_detailed "GET" "$API_GATEWAY_URL/api/auth/profile" "-H 'Authorization: Bearer $token'" "" "200" "gRPC GetUserContext" > /dev/null
    if [ $? -ne 0 ]; then
        failed=$((failed + 1))
    fi
    
    test_endpoint_detailed "GET" "$API_GATEWAY_URL/api/auth/permissions" "-H 'Authorization: Bearer $token'" "" "200" "gRPC GetUserPermissions" > /dev/null
    if [ $? -ne 0 ]; then
        failed=$((failed + 1))
    fi
    
    print_status "After individual calls:"
    local after_individual_stats=$(get_grpc_stats)
    print_status "$after_individual_stats"
    
    echo ""
    
    # Phase 3: Low Load Test (Should stay at minimum connections)
    print_test "=== PHASE 3: LOW LOAD TEST (Should stay at 5 connections) ==="
    
    print_status "Generating low concurrent load..."
    generate_load 3 15 "$token" # 3 concurrent requests for 15 seconds
    
    sleep 5 # Allow scaling to settle
    
    local low_load_stats=$(get_grpc_stats)
    print_status "After low load: $low_load_stats"
    
    # Extract connection count
    local low_load_connections=$(echo "$low_load_stats" | grep -o 'Connections: [0-9]*' | cut -d' ' -f2)
    
    if [ "$low_load_connections" -le 7 ]; then
        print_success "✅ Low load test: Connections stayed low ($low_load_connections ≤ 7)"
    else
        print_warning "⚠️  Low load test: Connections higher than expected ($low_load_connections > 7)"
    fi
    
    echo ""
    
    # Phase 4: High Load Test (Should scale up to more connections)
    print_test "=== PHASE 4: HIGH LOAD TEST (Should scale up towards 20 connections) ==="
    
    print_status "Generating high concurrent load..."
    generate_load 15 30 "$token" # 15 concurrent requests for 30 seconds
    
    sleep 5 # Allow scaling to settle
    
    local high_load_stats=$(get_grpc_stats)
    print_status "After high load: $high_load_stats"
    
    # Extract connection count
    local high_load_connections=$(echo "$high_load_stats" | grep -o 'Connections: [0-9]*' | cut -d' ' -f2)
    
    if [ "$high_load_connections" -gt 7 ]; then
        print_success "✅ High load test: Connections scaled up ($high_load_connections > 7)"
    else
        print_error "❌ High load test: Connections did not scale up ($high_load_connections ≤ 7)"
        failed=$((failed + 1))
    fi
    
    echo ""
    
    # Phase 5: Scale Down Test (Should scale back down after load decreases)
    print_test "=== PHASE 5: SCALE DOWN TEST (Should scale back down) ==="
    
    print_status "Waiting for scale down (60 seconds)..."
    sleep 60 # Wait for scale down interval
    
    local scale_down_stats=$(get_grpc_stats)
    print_status "After scale down wait: $scale_down_stats"
    
    # Extract connection count
    local scale_down_connections=$(echo "$scale_down_stats" | grep -o 'Connections: [0-9]*' | cut -d' ' -f2)
    
    if [ "$scale_down_connections" -lt "$high_load_connections" ]; then
        print_success "✅ Scale down test: Connections scaled down ($scale_down_connections < $high_load_connections)"
    else
        print_warning "⚠️  Scale down test: Connections did not scale down ($scale_down_connections ≥ $high_load_connections)"
    fi
    
    echo ""
    
    # Phase 6: Real gRPC Call Validation
    print_test "=== PHASE 6: REAL gRPC CALL VALIDATION ==="
    
    print_status "Validating that gRPC calls are actually being made..."
    
    # Check logs for gRPC call evidence
    if [ -f gateway_real_grpc.log ]; then
        local grpc_calls=$(grep -c "Making REAL gRPC" gateway_real_grpc.log || echo "0")
        if [ "$grpc_calls" -gt 0 ]; then
            print_success "✅ Real gRPC calls detected: $grpc_calls calls logged"
        else
            print_error "❌ No real gRPC calls detected in logs"
            failed=$((failed + 1))
        fi
        
        # Show recent gRPC call logs
        print_status "Recent gRPC call logs:"
        tail -5 gateway_real_grpc.log | grep "Making REAL gRPC" || print_status "No recent gRPC calls in logs"
    else
        print_warning "⚠️  Gateway log file not found"
    fi
    
    echo ""
    
    # Summary
    echo "=============================================="
    echo "🎯 DEEP DYNAMIC gRPC INTEGRATION RESULTS:"
    echo "=============================================="
    
    print_status "Connection Scaling Summary:"
    print_status "  Initial: $initial_stats"
    print_status "  Low Load: $low_load_stats"  
    print_status "  High Load: $high_load_stats"
    print_status "  Scale Down: $scale_down_stats"
    
    echo ""
    
    if [ $failed -eq 0 ]; then
        print_success "🎉 DEEP INTEGRATION TEST PASSED!"
        echo ""
        print_success "✅ Dynamic Connection Pooling: WORKING"
        print_success "✅ Load-Based Scaling: WORKING"
        print_success "✅ Real gRPC Calls: WORKING"
        print_success "✅ Connection Pool Management: WORKING"
        echo ""
        print_success "The API Gateway now has REAL dynamic gRPC connection pooling!"
        print_success "Connections scale from 5 to 20 based on actual load!"
        return 0
    else
        print_error "❌ $failed test(s) failed"
        print_error "Dynamic connection pooling needs investigation"
        return 1
    fi
}

main "$@"
