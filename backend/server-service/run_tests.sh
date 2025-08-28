#!/bin/bash

# API Gateway Test Suite Runner
# This script runs comprehensive tests for the API Gateway

set -e

echo "🚀 API Gateway Test Suite"
echo "========================="

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Check if Go is installed
if ! command -v go &> /dev/null; then
    print_error "Go is not installed or not in PATH"
    exit 1
fi

print_status "Go version: $(go version)"

# Set test timeout
TEST_TIMEOUT="30s"

# Create test results directory
mkdir -p test_results

# Function to run tests with coverage
run_tests() {
    local test_name=$1
    local test_path=$2
    local coverage_file="test_results/${test_name}_coverage.out"
    
    print_status "Running $test_name tests..."
    
    if go test -v -timeout=$TEST_TIMEOUT -coverprofile="$coverage_file" "$test_path" 2>&1 | tee "test_results/${test_name}_results.txt"; then
        print_success "$test_name tests passed"
        
        # Generate coverage report
        if [ -f "$coverage_file" ]; then
            coverage=$(go tool cover -func="$coverage_file" | grep total | awk '{print $3}')
            print_status "$test_name coverage: $coverage"
        fi
        
        return 0
    else
        print_error "$test_name tests failed"
        return 1
    fi
}

# Function to run benchmarks
run_benchmarks() {
    local test_path=$1
    
    print_status "Running benchmarks..."
    
    if go test -bench=. -benchmem "$test_path" 2>&1 | tee "test_results/benchmark_results.txt"; then
        print_success "Benchmarks completed"
        return 0
    else
        print_warning "Benchmarks failed or not available"
        return 1
    fi
}

# Function to check code formatting
check_formatting() {
    print_status "Checking code formatting..."
    
    unformatted=$(gofmt -l .)
    if [ -z "$unformatted" ]; then
        print_success "Code is properly formatted"
        return 0
    else
        print_error "The following files are not properly formatted:"
        echo "$unformatted"
        return 1
    fi
}

# Function to run static analysis
run_static_analysis() {
    print_status "Running static analysis..."
    
    # Check if golint is available
    if command -v golint &> /dev/null; then
        golint ./... > test_results/lint_results.txt 2>&1
        if [ -s test_results/lint_results.txt ]; then
            print_warning "Linting issues found (see test_results/lint_results.txt)"
        else
            print_success "No linting issues found"
        fi
    else
        print_warning "golint not available, skipping lint check"
    fi
    
    # Run go vet
    if go vet ./... 2>&1 | tee test_results/vet_results.txt; then
        print_success "go vet passed"
    else
        print_error "go vet found issues"
        return 1
    fi
}

# Function to check dependencies
check_dependencies() {
    print_status "Checking dependencies..."
    
    if go mod verify; then
        print_success "Dependencies verified"
    else
        print_error "Dependency verification failed"
        return 1
    fi
    
    if go mod tidy -diff; then
        print_success "go.mod is tidy"
    else
        print_warning "go.mod needs tidying"
    fi
}

# Main test execution
main() {
    local exit_code=0
    
    print_status "Starting comprehensive test suite..."
    
    # Check dependencies first
    if ! check_dependencies; then
        exit_code=1
    fi
    
    # Check code formatting
    if ! check_formatting; then
        exit_code=1
    fi
    
    # Run static analysis
    if ! run_static_analysis; then
        exit_code=1
    fi
    
    # Run unit tests
    print_status "Running unit tests..."
    
    # Test circuit breaker
    if ! run_tests "circuit_breaker" "./test"; then
        exit_code=1
    fi
    
    # Test error handler
    if ! run_tests "error_handler" "./test"; then
        exit_code=1
    fi
    
    # Test individual packages
    packages=(
        "./internal/circuit"
        "./internal/errors"
        "./internal/middleware"
        "./internal/router"
        "./internal/services"
    )
    
    for package in "${packages[@]}"; do
        if [ -d "$package" ]; then
            package_name=$(basename "$package")
            if ! run_tests "$package_name" "$package"; then
                exit_code=1
            fi
        fi
    done
    
    # Run integration tests
    print_status "Running integration tests..."
    if ! run_tests "integration" "./test"; then
        exit_code=1
    fi
    
    # Run benchmarks (optional)
    run_benchmarks "./test" || true
    
    # Generate combined coverage report
    print_status "Generating combined coverage report..."
    if ls test_results/*_coverage.out 1> /dev/null 2>&1; then
        echo "mode: set" > test_results/combined_coverage.out
        grep -h -v "mode: set" test_results/*_coverage.out >> test_results/combined_coverage.out
        
        total_coverage=$(go tool cover -func=test_results/combined_coverage.out | grep total | awk '{print $3}')
        print_status "Total coverage: $total_coverage"
        
        # Generate HTML coverage report
        go tool cover -html=test_results/combined_coverage.out -o test_results/coverage.html
        print_status "HTML coverage report generated: test_results/coverage.html"
    fi
    
    # Summary
    echo ""
    echo "========================="
    if [ $exit_code -eq 0 ]; then
        print_success "All tests passed! ✅"
    else
        print_error "Some tests failed! ❌"
    fi
    echo "========================="
    
    # Test results summary
    print_status "Test results saved in test_results/ directory"
    print_status "Available reports:"
    ls -la test_results/
    
    exit $exit_code
}

# Handle script arguments
case "${1:-}" in
    "unit")
        print_status "Running unit tests only..."
        run_tests "circuit_breaker" "./test"
        run_tests "error_handler" "./test"
        ;;
    "integration")
        print_status "Running integration tests only..."
        run_tests "integration" "./test"
        ;;
    "coverage")
        print_status "Running tests with coverage focus..."
        main
        ;;
    "quick")
        print_status "Running quick test suite..."
        check_formatting
        run_tests "circuit_breaker" "./test"
        run_tests "error_handler" "./test"
        ;;
    *)
        main
        ;;
esac
