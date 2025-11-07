#!/bin/bash
# Profile Analysis Helper Script
# This script helps analyze Go profiling results

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Default values
PROFILE_DIR="./profiles"
PORT=8080

# Print usage
usage() {
    echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║           Go Profile Analysis Helper                          ║${NC}"
    echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}"
    echo
    echo "Usage: $0 [command] [options]"
    echo
    echo "Commands:"
    echo "  list              List all available profile files"
    echo "  cpu [file]        Analyze CPU profile (latest if no file specified)"
    echo "  mem [file]        Analyze memory profile (latest if no file specified)"
    echo "  web [type] [file] Open profile in web browser (type: cpu|mem)"
    echo "  compare [file1] [file2]  Compare two CPU profiles"
    echo "  timings [file]    Show function timing report"
    echo "  top [file]        Show top functions from CPU profile"
    echo
    echo "Options:"
    echo "  -d, --dir DIR     Profile directory (default: ./profiles)"
    echo "  -p, --port PORT   Web server port (default: 8080)"
    echo "  -h, --help        Show this help message"
    echo
    echo "Examples:"
    echo "  $0 list"
    echo "  $0 cpu"
    echo "  $0 web cpu"
    echo "  $0 top profiles/cpu_20250113_143022.prof"
    echo "  $0 compare profiles/cpu_before.prof profiles/cpu_after.prof"
    echo
}

# Check if go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Error: Go is not installed${NC}"
        exit 1
    fi
}

# Get latest profile file of a type
get_latest_profile() {
    local type=$1
    local latest=$(ls -t ${PROFILE_DIR}/${type}_*.prof 2>/dev/null | head -1)
    if [ -z "$latest" ]; then
        echo -e "${RED}Error: No ${type} profile found in ${PROFILE_DIR}${NC}"
        exit 1
    fi
    echo "$latest"
}

# List all profile files
list_profiles() {
    echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${CYAN}║              Available Profile Files                          ║${NC}"
    echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}"
    echo

    if [ ! -d "$PROFILE_DIR" ]; then
        echo -e "${YELLOW}No profile directory found at: ${PROFILE_DIR}${NC}"
        echo -e "${YELLOW}Run your program with --profile flag to generate profiles${NC}"
        exit 0
    fi

    local cpu_files=$(ls -t ${PROFILE_DIR}/cpu_*.prof 2>/dev/null)
    local mem_files=$(ls -t ${PROFILE_DIR}/mem_*.prof 2>/dev/null)
    local timing_files=$(ls -t ${PROFILE_DIR}/timings_*.txt 2>/dev/null)

    echo -e "${GREEN}CPU Profiles:${NC}"
    if [ -z "$cpu_files" ]; then
        echo "  (none)"
    else
        echo "$cpu_files" | nl -w2 -s'. '
    fi
    echo

    echo -e "${GREEN}Memory Profiles:${NC}"
    if [ -z "$mem_files" ]; then
        echo "  (none)"
    else
        echo "$mem_files" | nl -w2 -s'. '
    fi
    echo

    echo -e "${GREEN}Timing Reports:${NC}"
    if [ -z "$timing_files" ]; then
        echo "  (none)"
    else
        echo "$timing_files" | nl -w2 -s'. '
    fi
    echo
}

# Analyze CPU profile
analyze_cpu() {
    local file=${1:-$(get_latest_profile "cpu")}

    if [ ! -f "$file" ]; then
        echo -e "${RED}Error: Profile file not found: ${file}${NC}"
        exit 1
    fi

    echo -e "${CYAN}Analyzing CPU Profile: ${file}${NC}"
    echo
    go tool pprof -top -nodecount=20 "$file"
}

# Analyze memory profile
analyze_mem() {
    local file=${1:-$(get_latest_profile "mem")}

    if [ ! -f "$file" ]; then
        echo -e "${RED}Error: Profile file not found: ${file}${NC}"
        exit 1
    fi

    echo -e "${CYAN}Analyzing Memory Profile: ${file}${NC}"
    echo
    echo -e "${YELLOW}Top allocations:${NC}"
    go tool pprof -top -nodecount=20 "$file"
    echo
    echo -e "${YELLOW}Allocation by function:${NC}"
    go tool pprof -list="." "$file" | head -50
}

# Open profile in web browser
open_web() {
    local type=$1
    local file=${2:-$(get_latest_profile "$type")}

    if [ ! -f "$file" ]; then
        echo -e "${RED}Error: Profile file not found: ${file}${NC}"
        exit 1
    fi

    echo -e "${CYAN}Opening ${type} profile in web browser: ${file}${NC}"
    echo -e "${GREEN}Server running at: http://localhost:${PORT}${NC}"
    echo -e "${YELLOW}Press Ctrl+C to stop${NC}"
    echo
    go tool pprof -http=:${PORT} "$file"
}

# Show top functions
show_top() {
    local file=${1:-$(get_latest_profile "cpu")}

    if [ ! -f "$file" ]; then
        echo -e "${RED}Error: Profile file not found: ${file}${NC}"
        exit 1
    fi

    echo -e "${CYAN}Top functions by CPU usage: ${file}${NC}"
    echo
    go tool pprof -top -nodecount=30 "$file"
}

# Compare two profiles
compare_profiles() {
    local file1=$1
    local file2=$2

    if [ -z "$file1" ] || [ -z "$file2" ]; then
        echo -e "${RED}Error: Two profile files required for comparison${NC}"
        usage
        exit 1
    fi

    if [ ! -f "$file1" ] || [ ! -f "$file2" ]; then
        echo -e "${RED}Error: One or both profile files not found${NC}"
        exit 1
    fi

    echo -e "${CYAN}Comparing profiles:${NC}"
    echo -e "  Base:   ${file1}"
    echo -e "  New:    ${file2}"
    echo
    echo -e "${YELLOW}Difference (negative means improvement):${NC}"
    echo
    go tool pprof -base="$file1" -top "$file2"
}

# Show timing report
show_timings() {
    local file=${1:-$(ls -t ${PROFILE_DIR}/timings_*.txt 2>/dev/null | head -1)}

    if [ -z "$file" ] || [ ! -f "$file" ]; then
        echo -e "${RED}Error: Timing report not found${NC}"
        exit 1
    fi

    echo -e "${CYAN}Function Timing Report: ${file}${NC}"
    echo
    cat "$file"
}

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        -d|--dir)
            PROFILE_DIR="$2"
            shift 2
            ;;
        -p|--port)
            PORT="$2"
            shift 2
            ;;
        -h|--help)
            usage
            exit 0
            ;;
        list)
            check_go
            list_profiles
            exit 0
            ;;
        cpu)
            check_go
            analyze_cpu "$2"
            exit 0
            ;;
        mem)
            check_go
            analyze_mem "$2"
            exit 0
            ;;
        web)
            check_go
            open_web "$2" "$3"
            exit 0
            ;;
        top)
            check_go
            show_top "$2"
            exit 0
            ;;
        compare)
            check_go
            compare_profiles "$2" "$3"
            exit 0
            ;;
        timings)
            show_timings "$2"
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown command: $1${NC}"
            echo
            usage
            exit 1
            ;;
    esac
done

# No command provided
usage
