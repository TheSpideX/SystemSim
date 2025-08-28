#!/bin/bash

# Focused Dynamic Scaling Test
# Tests if gRPC connection pool actually scales under load

set -e

API_GATEWAY_URL="https://localhost:8000"

# Get fresh token
echo "Getting authentication token..."
RESPONSE=$(curl -k -s https://localhost:8000/api/auth/login -X POST -H "Content-Type: application/json" -d '{"email":"deep_test@example.com","password":"ComplexP@ssw0rd2024!"}')
TOKEN=$(echo "$RESPONSE" | grep -o '"access_token":"[^"]*"' | cut -d'"' -f4)

if [ -z "$TOKEN" ]; then
    echo "Failed to get token"
    exit 1
fi

echo "Token obtained: ${TOKEN:0:30}..."

# Function to get connection count
get_connections() {
    curl -k -s "$API_GATEWAY_URL/grpc/stats" | grep -o '"current_connections":[0-9]*' | cut -d':' -f2 | head -n1
}

# Function to get active requests
get_active() {
    curl -k -s "$API_GATEWAY_URL/grpc/stats" | grep -o '"active_requests":[0-9]*' | cut -d':' -f2 | head -n1
}

# Function to get utilization
get_utilization() {
    curl -k -s "$API_GATEWAY_URL/grpc/stats" | grep -o '"utilization":[0-9.]*' | cut -d':' -f2 | head -n1
}

echo ""
echo "=== INITIAL STATE ==="
INITIAL_CONNECTIONS=$(get_connections)
echo "Initial connections: $INITIAL_CONNECTIONS"

echo ""
echo "=== GENERATING HIGH LOAD ==="
echo "Starting 25 concurrent requests for 60 seconds..."

# Start 25 concurrent request generators
for i in {1..25}; do
    (
        count=0
        end_time=$(($(date +%s) + 60))
        while [ $(date +%s) -lt $end_time ]; do
            curl -k -s -X POST "$API_GATEWAY_URL/api/auth/validate" \
                -H "Authorization: Bearer $TOKEN" > /dev/null 2>&1
            count=$((count + 1))
            sleep 0.1  # 100ms between requests
        done
        echo "Thread $i: $count requests" > /tmp/load_result_$i.txt
    ) &
done

# Monitor scaling every 5 seconds during load
echo ""
echo "Time | Connections | Active | Utilization"
echo "-----|-------------|--------|------------"

for t in {5..60..5}; do
    sleep 5
    connections=$(get_connections)
    active=$(get_active)
    utilization=$(get_utilization)
    printf "%4ds | %11s | %6s | %s\n" "$t" "$connections" "$active" "$utilization"
done

# Wait for all background jobs to complete
wait

echo ""
echo "=== FINAL STATE ==="
FINAL_CONNECTIONS=$(get_connections)
FINAL_ACTIVE=$(get_active)
FINAL_UTILIZATION=$(get_utilization)

echo "Final connections: $FINAL_CONNECTIONS"
echo "Final active: $FINAL_ACTIVE"
echo "Final utilization: $FINAL_UTILIZATION"

# Calculate total requests
total_requests=0
for i in {1..25}; do
    if [ -f /tmp/load_result_$i.txt ]; then
        count=$(cat /tmp/load_result_$i.txt | grep -o '[0-9]*' | tail -1)
        total_requests=$((total_requests + count))
        rm -f /tmp/load_result_$i.txt
    fi
done

echo "Total requests made: $total_requests"

echo ""
echo "=== SCALING ANALYSIS ==="
if [ "$FINAL_CONNECTIONS" -gt "$INITIAL_CONNECTIONS" ]; then
    echo "✅ SCALING UP DETECTED: $INITIAL_CONNECTIONS → $FINAL_CONNECTIONS connections"
    echo "✅ Dynamic scaling is WORKING!"
else
    echo "❌ NO SCALING DETECTED: $INITIAL_CONNECTIONS → $FINAL_CONNECTIONS connections"
    echo "❌ Dynamic scaling is NOT working!"
    
    echo ""
    echo "Checking logs for scaling activity..."
    if grep -q "Scaled UP" gateway_debug_scaling.log; then
        echo "✅ Scale up logs found in gateway logs"
    else
        echo "❌ No scale up logs found"
    fi
    
    if grep -q "utilization.*[0-9]" gateway_debug_scaling.log; then
        echo "Recent utilization checks:"
        tail -20 gateway_debug_scaling.log | grep "utilization"
    fi
fi

echo ""
echo "=== WAITING FOR SCALE DOWN ==="
echo "Waiting 90 seconds for scale down..."
sleep 90

SCALE_DOWN_CONNECTIONS=$(get_connections)
echo "After scale down wait: $SCALE_DOWN_CONNECTIONS connections"

if [ "$SCALE_DOWN_CONNECTIONS" -lt "$FINAL_CONNECTIONS" ]; then
    echo "✅ SCALE DOWN DETECTED: $FINAL_CONNECTIONS → $SCALE_DOWN_CONNECTIONS connections"
else
    echo "⚠️  NO SCALE DOWN: $FINAL_CONNECTIONS → $SCALE_DOWN_CONNECTIONS connections (may need more time)"
fi

echo ""
echo "=== FINAL VERDICT ==="
if [ "$FINAL_CONNECTIONS" -gt "$INITIAL_CONNECTIONS" ]; then
    echo "🎉 DYNAMIC SCALING IS WORKING!"
    exit 0
else
    echo "🔧 DYNAMIC SCALING NEEDS INVESTIGATION"
    exit 1
fi
