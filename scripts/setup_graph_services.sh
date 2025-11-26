#!/bin/bash

# Archivist Knowledge Graph Services Setup Script
# Manages Neo4j, Qdrant, Redis, and Kafka services

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

# Functions
print_header() {
    echo -e "\n${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}  $1${NC}"
    echo -e "${MAGENTA}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_info() {
    echo -e "${BLUE}ℹ${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

# Banner
cat << "EOF"

╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║        Knowledge Graph Services Setup & Management           ║
║                                                               ║
║   Neo4j • Qdrant • Redis • Kafka                             ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝

EOF

# Check Docker
if ! command -v docker &> /dev/null; then
    print_error "Docker is not installed"
    echo "Please install Docker first: https://docs.docker.com/get-docker/"
    exit 1
fi

if ! docker info &> /dev/null; then
    print_error "Docker daemon is not running"
    echo "Please start Docker and try again"
    exit 1
fi

# Check for docker-compose
if command -v docker-compose &> /dev/null; then
    COMPOSE_CMD="docker-compose"
elif docker compose version &> /dev/null 2>&1; then
    COMPOSE_CMD="docker compose"
else
    print_error "Docker Compose is not installed"
    exit 1
fi

# Parse command
COMMAND=${1:-"start"}

case "$COMMAND" in
    start)
        print_header "Starting Knowledge Graph Services"

        # Check if docker-compose-graph.yml exists
        if [ ! -f "docker-compose-graph.yml" ]; then
            print_error "docker-compose-graph.yml not found"
            exit 1
        fi

        print_info "Starting Docker Compose services..."
        $COMPOSE_CMD -f docker-compose-graph.yml up -d

        print_info "Waiting for services to initialize..."
        sleep 5

        # Health checks
        print_header "Health Checks"

        # Neo4j
        print_info "Checking Neo4j..."
        for i in {1..30}; do
            if docker exec archivist-neo4j cypher-shell -u neo4j -p password "RETURN 1" &> /dev/null; then
                print_success "Neo4j is ready (http://localhost:7474)"
                break
            fi
            if [ $i -eq 30 ]; then
                print_warning "Neo4j health check timeout"
            fi
            sleep 2
        done

        # Qdrant
        print_info "Checking Qdrant..."
        for i in {1..30}; do
            if curl -s http://localhost:6333/healthz &> /dev/null; then
                print_success "Qdrant is ready (http://localhost:6333/dashboard)"
                break
            fi
            if [ $i -eq 30 ]; then
                print_warning "Qdrant health check timeout"
            fi
            sleep 2
        done

        # Redis
        print_info "Checking Redis..."
        for i in {1..30}; do
            if docker exec archivist-redis redis-cli ping &> /dev/null; then
                print_success "Redis is ready (localhost:6379)"
                break
            fi
            if [ $i -eq 30 ]; then
                print_warning "Redis health check timeout"
            fi
            sleep 2
        done

        # Kafka
        print_info "Checking Kafka..."
        for i in {1..30}; do
            if docker exec archivist-kafka kafka-broker-api-versions.sh --bootstrap-server localhost:9092 &> /dev/null; then
                print_success "Kafka is ready (localhost:9092)"
                break
            fi
            if [ $i -eq 30 ]; then
                print_warning "Kafka health check timeout"
            fi
            sleep 2
        done

        print_header "✅ All Services Started"

        cat << EOF

${GREEN}Access your services:${NC}

  📊 Neo4j Browser:       ${CYAN}http://localhost:7474${NC}
     Username: neo4j
     Password: password

  🔍 Qdrant Dashboard:    ${CYAN}http://localhost:6333/dashboard${NC}

  💾 Redis:               ${CYAN}localhost:6379${NC}

  📨 Kafka:               ${CYAN}localhost:9092${NC}

${YELLOW}Next Steps:${NC}

  1. Enable graph in config/config.yaml:
     ${CYAN}graph:
       enabled: true${NC}

  2. Process papers with graph building:
     ${CYAN}./archivist process lib/*.pdf --with-graph${NC}

  3. Search semantically:
     ${CYAN}./archivist search "transformer architecture"${NC}

  4. Explore citations:
     ${CYAN}./archivist cite show "Paper Title"${NC}

EOF
        ;;

    stop)
        print_header "Stopping Knowledge Graph Services"
        $COMPOSE_CMD -f docker-compose-graph.yml stop
        print_success "All services stopped"
        ;;

    down)
        print_header "Stopping and Removing Knowledge Graph Services"
        echo -e "${YELLOW}⚠️  This will remove containers but preserve data volumes${NC}"
        $COMPOSE_CMD -f docker-compose-graph.yml down
        print_success "Services removed (data preserved)"
        ;;

    restart)
        print_header "Restarting Knowledge Graph Services"
        $COMPOSE_CMD -f docker-compose-graph.yml restart
        print_success "Services restarted"
        ;;

    status)
        print_header "Service Status"
        $COMPOSE_CMD -f docker-compose-graph.yml ps
        ;;

    logs)
        SERVICE=${2:-""}
        if [ -z "$SERVICE" ]; then
            print_info "Showing logs for all services (Ctrl+C to exit)"
            $COMPOSE_CMD -f docker-compose-graph.yml logs -f
        else
            print_info "Showing logs for $SERVICE (Ctrl+C to exit)"
            $COMPOSE_CMD -f docker-compose-graph.yml logs -f $SERVICE
        fi
        ;;

    clean)
        print_header "Cleaning Knowledge Graph Data"
        echo -e "${RED}⚠️  WARNING: This will DELETE ALL DATA from the knowledge graph!${NC}"
        echo -e "This includes:"
        echo -e "  • All papers in Neo4j"
        echo -e "  • All vectors in Qdrant"
        echo -e "  • All cached data in Redis"
        echo -e "  • All Kafka messages"
        echo ""
        read -p "Are you sure? Type 'yes' to confirm: " confirm

        if [ "$confirm" = "yes" ]; then
            print_info "Stopping services..."
            $COMPOSE_CMD -f docker-compose-graph.yml down -v
            print_success "All data cleaned"
        else
            print_info "Clean cancelled"
        fi
        ;;

    reset)
        print_header "Resetting Knowledge Graph Services"
        echo -e "${RED}⚠️  This will clean data and restart services${NC}"
        read -p "Continue? [y/N]: " confirm

        if [[ "$confirm" =~ ^[Yy]$ ]]; then
            print_info "Stopping and cleaning..."
            $COMPOSE_CMD -f docker-compose-graph.yml down -v

            print_info "Restarting services..."
            $COMPOSE_CMD -f docker-compose-graph.yml up -d

            print_success "Services reset complete"
        else
            print_info "Reset cancelled"
        fi
        ;;

    backup)
        BACKUP_DIR="./backups/graph-$(date +%Y%m%d-%H%M%S)"
        mkdir -p "$BACKUP_DIR"

        print_header "Backing Up Knowledge Graph Data"

        # Neo4j backup
        print_info "Backing up Neo4j..."
        docker exec archivist-neo4j neo4j-admin database dump neo4j --to-stdout > "$BACKUP_DIR/neo4j-dump.dump" 2>/dev/null || print_warning "Neo4j backup failed"

        # Qdrant backup
        print_info "Backing up Qdrant..."
        docker exec archivist-qdrant tar czf - /qdrant/storage > "$BACKUP_DIR/qdrant-storage.tar.gz" 2>/dev/null || print_warning "Qdrant backup failed"

        # Redis backup
        print_info "Backing up Redis..."
        docker exec archivist-redis redis-cli --rdb - > "$BACKUP_DIR/redis-dump.rdb" 2>/dev/null || print_warning "Redis backup failed"

        print_success "Backup saved to: $BACKUP_DIR"
        ;;

    test)
        print_header "Testing Knowledge Graph Services"

        # Test Neo4j
        print_info "Testing Neo4j connection..."
        if docker exec archivist-neo4j cypher-shell -u neo4j -p password "RETURN 'Neo4j OK' AS status" 2>/dev/null | grep -q "Neo4j OK"; then
            print_success "Neo4j: Connected"
        else
            print_error "Neo4j: Connection failed"
        fi

        # Test Qdrant
        print_info "Testing Qdrant connection..."
        if curl -s http://localhost:6333/healthz | grep -q "ok"; then
            print_success "Qdrant: Connected"
        else
            print_error "Qdrant: Connection failed"
        fi

        # Test Redis
        print_info "Testing Redis connection..."
        if docker exec archivist-redis redis-cli ping 2>/dev/null | grep -q "PONG"; then
            print_success "Redis: Connected"
        else
            print_error "Redis: Connection failed"
        fi

        # Test Kafka
        print_info "Testing Kafka connection..."
        if docker exec archivist-kafka kafka-broker-api-versions.sh --bootstrap-server localhost:9092 &>/dev/null; then
            print_success "Kafka: Connected"
        else
            print_error "Kafka: Connection failed"
        fi
        ;;

    help|--help|-h)
        cat << EOF
Knowledge Graph Services Manager

${CYAN}Usage:${NC}
  ./scripts/setup_graph_services.sh [command]

${CYAN}Commands:${NC}
  ${GREEN}start${NC}       Start all graph services (default)
  ${GREEN}stop${NC}        Stop all services
  ${GREEN}down${NC}        Stop and remove containers (keeps data)
  ${GREEN}restart${NC}     Restart all services
  ${GREEN}status${NC}      Show service status
  ${GREEN}logs${NC}        Show logs (optional: specify service name)
  ${GREEN}clean${NC}       Remove all data (destructive!)
  ${GREEN}reset${NC}       Clean data and restart services
  ${GREEN}backup${NC}      Backup all graph data
  ${GREEN}test${NC}        Test all service connections
  ${GREEN}help${NC}        Show this help message

${CYAN}Examples:${NC}
  # Start services
  ./scripts/setup_graph_services.sh start

  # View logs for Neo4j only
  ./scripts/setup_graph_services.sh logs neo4j

  # Clean and start fresh
  ./scripts/setup_graph_services.sh clean
  ./scripts/setup_graph_services.sh start

  # Backup before major changes
  ./scripts/setup_graph_services.sh backup

${CYAN}Service Ports:${NC}
  Neo4j:    7474 (HTTP), 7687 (Bolt)
  Qdrant:   6333 (HTTP), 6334 (gRPC)
  Redis:    6380 (mapped from 6379)
  Kafka:    9092 (Internal), 9094 (External)

EOF
        ;;

    *)
        print_error "Unknown command: $COMMAND"
        echo "Run './scripts/setup_graph_services.sh help' for usage"
        exit 1
        ;;
esac
