#!/bin/bash

# Archivist Test Runner Script
# This script provides comprehensive testing capabilities for the Archivist project

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print colored output
print_header() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

# Show usage
show_usage() {
    cat << EOF
Archivist Test Runner

Usage: ./test.sh [COMMAND] [OPTIONS]

Commands:
    all             Run all tests (unit + integration)
    unit            Run unit tests only
    integration     Run integration tests only
    coverage        Run tests with coverage report
    bench           Run benchmarks
    quick           Run quick tests for development
    verbose         Run tests with verbose output
    specific TEST   Run a specific test by name
    watch           Watch for changes and re-run tests (requires entr)
    clean           Clean test artifacts
    help            Show this help message

Options:
    -v, --verbose   Enable verbose output
    -r, --race      Enable race detector (default: enabled)
    -t, --timeout   Set timeout (default: 5m)

Examples:
    ./test.sh all
    ./test.sh unit --verbose
    ./test.sh specific TestComputeFileHash
    ./test.sh coverage
    ./test.sh watch

EOF
}

# Run all tests
run_all_tests() {
    print_header "Running All Tests"
    go test -race -timeout 5m ./...
    print_success "All tests passed!"
}

# Run unit tests
run_unit_tests() {
    print_header "Running Unit Tests"
    go test -race -short -timeout 2m ./pkg/... ./internal/storage ./internal/parser ./internal/generator ./internal/compiler ./internal/analyzer
    print_success "Unit tests passed!"
}

# Run integration tests
run_integration_tests() {
    print_header "Running Integration Tests"
    go test -race -timeout 5m ./internal -run Test.*Workflow
    print_success "Integration tests passed!"
}

# Run tests with coverage
run_coverage() {
    print_header "Running Tests with Coverage"
    go test -race -timeout 5m -coverprofile=coverage.out -covermode=atomic ./...

    print_info "Generating HTML coverage report..."
    go tool cover -html=coverage.out -o coverage.html

    print_info "Coverage summary:"
    go tool cover -func=coverage.out | grep total

    print_success "Coverage report generated: coverage.html"

    # Open coverage in browser if possible
    if command -v xdg-open &> /dev/null; then
        print_info "Opening coverage report in browser..."
        xdg-open coverage.html &> /dev/null &
    fi
}

# Run benchmarks
run_benchmarks() {
    print_header "Running Benchmarks"
    go test -bench=. -benchmem -run=^$$ ./...
    print_success "Benchmarks complete!"
}

# Run quick tests
run_quick_tests() {
    print_header "Running Quick Tests"
    go test -short -timeout 1m ./pkg/... ./internal/storage ./internal/generator
    print_success "Quick tests passed!"
}

# Run verbose tests
run_verbose_tests() {
    print_header "Running Tests (Verbose)"
    go test -v -race -timeout 5m ./...
}

# Run specific test
run_specific_test() {
    TEST_NAME=$1
    if [ -z "$TEST_NAME" ]; then
        print_error "Test name required"
        echo "Usage: ./test.sh specific TestName"
        exit 1
    fi

    print_header "Running Test: $TEST_NAME"
    go test -v -race -run "$TEST_NAME" ./...
}

# Watch mode - rerun tests on changes
run_watch() {
    if ! command -v entr &> /dev/null; then
        print_error "entr not installed. Install with: sudo apt install entr"
        exit 1
    fi

    print_header "Watching for changes..."
    print_info "Press Ctrl+C to stop"

    find . -name '*.go' | entr -c bash -c './test.sh quick'
}

# Clean test artifacts
clean_artifacts() {
    print_header "Cleaning Test Artifacts"
    rm -f coverage.out coverage.html
    print_success "Test artifacts cleaned!"
}

# Check prerequisites
check_prerequisites() {
    print_info "Checking prerequisites..."

    # Check Go installation
    if ! command -v go &> /dev/null; then
        print_error "Go is not installed"
        exit 1
    fi

    GO_VERSION=$(go version | awk '{print $3}')
    print_success "Go is installed: $GO_VERSION"

    # Check if in correct directory
    if [ ! -f "go.mod" ]; then
        print_error "Not in project root directory"
        exit 1
    fi

    print_success "Prerequisites check passed!"
}

# Main script
main() {
    case "${1:-help}" in
        all)
            check_prerequisites
            run_all_tests
            ;;
        unit)
            check_prerequisites
            run_unit_tests
            ;;
        integration)
            check_prerequisites
            run_integration_tests
            ;;
        coverage)
            check_prerequisites
            run_coverage
            ;;
        bench|benchmark)
            check_prerequisites
            run_benchmarks
            ;;
        quick)
            check_prerequisites
            run_quick_tests
            ;;
        verbose)
            check_prerequisites
            run_verbose_tests
            ;;
        specific)
            check_prerequisites
            run_specific_test "$2"
            ;;
        watch)
            check_prerequisites
            run_watch
            ;;
        clean)
            clean_artifacts
            ;;
        help|--help|-h)
            show_usage
            ;;
        *)
            print_error "Unknown command: $1"
            show_usage
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
