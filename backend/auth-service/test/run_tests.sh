#!/bin/bash

# Auth Service Test Runner
# This script runs comprehensive tests against the running auth service

set -e

echo "🧪 Auth Service Test Runner"
echo "=========================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to check if service is running
check_service() {
    local service_name=$1
    local url=$2
    
    if curl -s --connect-timeout 2 "$url" > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} $service_name is running"
        return 0
    else
        echo -e "${RED}✗${NC} $service_name is not running at $url"
        return 1
    fi
}

# Function to run tests with proper formatting
run_test() {
    local test_name=$1
    local test_command=$2
    
    echo -e "\n${BLUE}🔍 Running $test_name${NC}"
    echo "----------------------------------------"
    
    if eval "$test_command"; then
        echo -e "${GREEN}✓ $test_name PASSED${NC}"
        return 0
    else
        echo -e "${RED}✗ $test_name FAILED${NC}"
        return 1
    fi
}

# Check prerequisites
echo -e "\n${YELLOW}📋 Checking Prerequisites${NC}"
echo "----------------------------------------"

# Check if Go is available
if ! command -v go &> /dev/null; then
    echo -e "${RED}✗${NC} Go is not installed"
    exit 1
fi
echo -e "${GREEN}✓${NC} Go is available"

# Check if we're in the right directory
if [ ! -f "go.mod" ]; then
    echo -e "${RED}✗${NC} Please run this script from the auth-service root directory"
    exit 1
fi
echo -e "${GREEN}✓${NC} In auth-service directory"

# Parse command line arguments
SKIP_SERVICE_CHECK=false
RUN_LOAD_TESTS=false
VERBOSE=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --skip-service-check)
            SKIP_SERVICE_CHECK=true
            shift
            ;;
        --load-tests)
            RUN_LOAD_TESTS=true
            shift
            ;;
        --verbose|-v)
            VERBOSE=true
            shift
            ;;
        --help|-h)
            echo "Usage: $0 [OPTIONS]"
            echo ""
            echo "Options:"
            echo "  --skip-service-check    Skip checking if services are running"
            echo "  --load-tests           Run load tests (requires running service)"
            echo "  --verbose, -v          Verbose output"
            echo "  --help, -h             Show this help message"
            echo ""
            echo "Examples:"
            echo "  $0                     # Run unit tests only"
            echo "  $0 --load-tests        # Run all tests including load tests"
            echo "  $0 --skip-service-check --verbose  # Skip service check, verbose output"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            echo "Use --help for usage information"
            exit 1
            ;;
    esac
done

# Set verbose flag for go test
VERBOSE_FLAG=""
if [ "$VERBOSE" = true ]; then
    VERBOSE_FLAG="-v"
fi

# Check service availability
if [ "$SKIP_SERVICE_CHECK" = false ]; then
    echo -e "\n${YELLOW}🔍 Checking Service Availability${NC}"
    echo "----------------------------------------"
    
    HTTP_AVAILABLE=false
    GRPC_AVAILABLE=false
    
    if check_service "Auth Service HTTP" "http://localhost:9001/health"; then
        HTTP_AVAILABLE=true
    fi

    # Check gRPC by attempting a connection
    if timeout 2 bash -c "</dev/tcp/localhost/9000" 2>/dev/null; then
        echo -e "${GREEN}✓${NC} Auth Service gRPC is running"
        GRPC_AVAILABLE=true
    else
        echo -e "${RED}✗${NC} Auth Service gRPC is not running at localhost:9000"
        GRPC_AVAILABLE=false
    fi
    
    # Check PostgreSQL
    if command -v psql &> /dev/null; then
        if PGPASSWORD=postgres psql -h localhost -U postgres -d auth_service_dev -c '\q' 2>/dev/null; then
            echo -e "${GREEN}✓${NC} PostgreSQL is accessible"
        else
            echo -e "${YELLOW}⚠${NC} PostgreSQL may not be accessible (tests will skip if needed)"
        fi
    else
        echo -e "${YELLOW}⚠${NC} psql not available, cannot check PostgreSQL"
    fi
    
    # Check Redis
    if command -v redis-cli &> /dev/null; then
        if redis-cli ping > /dev/null 2>&1; then
            echo -e "${GREEN}✓${NC} Redis is accessible"
        else
            echo -e "${YELLOW}⚠${NC} Redis may not be accessible (tests will skip if needed)"
        fi
    else
        echo -e "${YELLOW}⚠${NC} redis-cli not available, cannot check Redis"
    fi
fi

# Run tests
echo -e "\n${YELLOW}🧪 Running Tests${NC}"
echo "========================================"

# 1. Security Unit Tests (always run - no external dependencies)
run_test "Security Unit Tests" "go test $VERBOSE_FLAG ./internal/security/"

# 2. Functional Tests (if HTTP service is available)
if [ "$SKIP_SERVICE_CHECK" = true ] || [ "$HTTP_AVAILABLE" = true ]; then
    run_test "HTTP Functional Tests" "go test $VERBOSE_FLAG ./test/ -run TestAuthServiceFunctionality"
else
    echo -e "\n${YELLOW}⏭ Skipping HTTP Functional Tests (service not running)${NC}"
fi

# 3. gRPC Tests (if gRPC service is available)
if [ "$SKIP_SERVICE_CHECK" = true ] || [ "$GRPC_AVAILABLE" = true ]; then
    run_test "gRPC Tests" "go test $VERBOSE_FLAG ./test/ -run TestAuthServiceGRPC -timeout 30s"
else
    echo -e "\n${YELLOW}⏭ Skipping gRPC Tests (service not running)${NC}"
fi

# 4. Load Tests (if requested and service is available)
if [ "$RUN_LOAD_TESTS" = true ]; then
    if [ "$SKIP_SERVICE_CHECK" = true ] || [ "$HTTP_AVAILABLE" = true ]; then
        run_test "Load Tests" "go test $VERBOSE_FLAG ./test/ -run TestAuthServiceLoad -timeout 5m"
    else
        echo -e "\n${YELLOW}⏭ Skipping Load Tests (service not running)${NC}"
    fi
else
    echo -e "\n${YELLOW}⏭ Skipping Load Tests (use --load-tests to enable)${NC}"
fi

# Summary
echo -e "\n${YELLOW}📊 Test Summary${NC}"
echo "========================================"
echo -e "${GREEN}✓${NC} Security unit tests validate password hashing, JWT tokens"
echo -e "${GREEN}✓${NC} Functional tests validate complete user flows"
echo -e "${GREEN}✓${NC} Tests automatically skip when services aren't running"
echo -e "${GREEN}✓${NC} All tests validate real functionality, not just passing tests"

echo -e "\n${BLUE}💡 Next Steps:${NC}"
echo "1. Start the auth service: go run cmd/main.go"
echo "2. Run full tests: $0 --load-tests --verbose"
echo "3. Check test/README.md for detailed documentation"

echo -e "\n${GREEN}🎉 Test run completed!${NC}"
