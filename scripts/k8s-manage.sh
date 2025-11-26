#!/bin/bash

# Kubernetes Management Script for Archivist
# Manage, scale, and monitor Kubernetes deployments

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

NAMESPACE="archivist"

# Functions
print_header() {
    echo -e "\n${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}  $1${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}\n"
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

# Command
COMMAND=${1:-"help"}

case "$COMMAND" in
    status)
        print_header "Cluster Status"
        echo -e "${CYAN}Pods:${NC}"
        kubectl get pods -n $NAMESPACE
        echo ""
        echo -e "${CYAN}Services:${NC}"
        kubectl get svc -n $NAMESPACE
        echo ""
        echo -e "${CYAN}Deployments:${NC}"
        kubectl get deployments -n $NAMESPACE
        echo ""
        echo -e "${CYAN}StatefulSets:${NC}"
        kubectl get statefulsets -n $NAMESPACE
        echo ""
        echo -e "${CYAN}HPA Status:${NC}"
        kubectl get hpa -n $NAMESPACE
        ;;

    scale)
        DEPLOYMENT=$2
        REPLICAS=$3

        if [ -z "$DEPLOYMENT" ] || [ -z "$REPLICAS" ]; then
            print_error "Usage: $0 scale <deployment-name> <replicas>"
            print_info "Available deployments:"
            kubectl get deployments -n $NAMESPACE --no-headers | awk '{print "  - " $1}'
            exit 1
        fi

        print_header "Scaling $DEPLOYMENT to $REPLICAS replicas"
        kubectl scale deployment/$DEPLOYMENT --replicas=$REPLICAS -n $NAMESPACE
        print_success "Scaled $DEPLOYMENT to $REPLICAS replicas"
        ;;

    logs)
        POD_NAME=$2

        if [ -z "$POD_NAME" ]; then
            print_info "Available pods:"
            kubectl get pods -n $NAMESPACE --no-headers | awk '{print "  - " $1}'
            echo ""
            read -p "Enter pod name (or deployment name): " POD_NAME
        fi

        # Check if it's a deployment name
        if kubectl get deployment/$POD_NAME -n $NAMESPACE &> /dev/null; then
            print_header "Logs for deployment: $POD_NAME"
            kubectl logs -f deployment/$POD_NAME -n $NAMESPACE
        else
            print_header "Logs for pod: $POD_NAME"
            kubectl logs -f $POD_NAME -n $NAMESPACE
        fi
        ;;

    exec)
        POD_NAME=$2

        if [ -z "$POD_NAME" ]; then
            print_info "Available pods:"
            kubectl get pods -n $NAMESPACE --no-headers | awk '{print "  - " $1}'
            echo ""
            read -p "Enter pod name: " POD_NAME
        fi

        print_header "Executing shell in: $POD_NAME"
        kubectl exec -it $POD_NAME -n $NAMESPACE -- /bin/bash || kubectl exec -it $POD_NAME -n $NAMESPACE -- /bin/sh
        ;;

    port-forward)
        SERVICE=$2

        if [ -z "$SERVICE" ]; then
            echo -e "${CYAN}Available services:${NC}"
            echo "  1. neo4j     (7474:7474, 7687:7687)"
            echo "  2. qdrant    (6333:6333)"
            echo "  3. redis     (6379:6379)"
            echo "  4. kafka     (9092:9092)"
            echo "  5. search    (8000:8000)"
            echo "  6. graph     (8081:8081)"
            echo "  7. worker    (8080:8080)"
            echo ""
            read -p "Select service (1-7): " choice

            case $choice in
                1) SERVICE="neo4j" ;;
                2) SERVICE="qdrant" ;;
                3) SERVICE="redis" ;;
                4) SERVICE="kafka" ;;
                5) SERVICE="search" ;;
                6) SERVICE="graph" ;;
                7) SERVICE="worker" ;;
                *) print_error "Invalid choice"; exit 1 ;;
            esac
        fi

        case $SERVICE in
            neo4j)
                print_header "Port forwarding Neo4j"
                print_info "Neo4j Browser: http://localhost:7474"
                print_info "Bolt: bolt://localhost:7687"
                kubectl port-forward svc/neo4j-service 7474:7474 7687:7687 -n $NAMESPACE
                ;;
            qdrant)
                print_header "Port forwarding Qdrant"
                print_info "Qdrant Dashboard: http://localhost:6333/dashboard"
                kubectl port-forward svc/qdrant-service 6333:6333 -n $NAMESPACE
                ;;
            redis)
                print_header "Port forwarding Redis"
                print_info "Redis: localhost:6379"
                kubectl port-forward svc/redis-service 6379:6379 -n $NAMESPACE
                ;;
            kafka)
                print_header "Port forwarding Kafka"
                print_info "Kafka: localhost:9092"
                kubectl port-forward svc/kafka-service 9092:9092 -n $NAMESPACE
                ;;
            search)
                print_header "Port forwarding Search Service"
                print_info "API: http://localhost:8000"
                kubectl port-forward svc/search-service 8000:8000 -n $NAMESPACE
                ;;
            graph)
                print_header "Port forwarding Graph Service"
                print_info "API: http://localhost:8081"
                kubectl port-forward svc/graph-service 8081:8081 -n $NAMESPACE
                ;;
            worker)
                print_header "Port forwarding Worker Service"
                print_info "API: http://localhost:8080"
                kubectl port-forward svc/archivist-worker-service 8080:8080 -n $NAMESPACE
                ;;
            *)
                print_error "Unknown service: $SERVICE"
                exit 1
                ;;
        esac
        ;;

    restart)
        DEPLOYMENT=$2

        if [ -z "$DEPLOYMENT" ]; then
            print_error "Usage: $0 restart <deployment-name>"
            print_info "Available deployments:"
            kubectl get deployments -n $NAMESPACE --no-headers | awk '{print "  - " $1}'
            exit 1
        fi

        print_header "Restarting $DEPLOYMENT"
        kubectl rollout restart deployment/$DEPLOYMENT -n $NAMESPACE
        print_success "Restart initiated for $DEPLOYMENT"

        print_info "Watching rollout status..."
        kubectl rollout status deployment/$DEPLOYMENT -n $NAMESPACE
        ;;

    hpa)
        print_header "Horizontal Pod Autoscaler Status"
        kubectl get hpa -n $NAMESPACE

        echo ""
        print_info "Watch HPA in real-time:"
        print_info "  watch kubectl get hpa -n $NAMESPACE"
        ;;

    top)
        print_header "Resource Usage"
        echo -e "${CYAN}Pods:${NC}"
        kubectl top pods -n $NAMESPACE
        echo ""
        echo -e "${CYAN}Nodes:${NC}"
        kubectl top nodes
        ;;

    delete)
        print_warning "This will delete ALL Archivist resources!"
        read -p "Are you sure? Type 'yes' to confirm: " confirm

        if [ "$confirm" = "yes" ]; then
            print_header "Deleting All Resources"
            kubectl delete namespace $NAMESPACE
            print_success "All resources deleted"
        else
            print_info "Deletion cancelled"
        fi
        ;;

    update)
        print_header "Updating Deployments"
        kubectl apply -f k8s/base/
        print_success "All manifests applied"
        ;;

    backup)
        BACKUP_DIR="./backups/k8s-$(date +%Y%m%d-%H%M%S)"
        mkdir -p "$BACKUP_DIR"

        print_header "Backing Up Kubernetes Resources"

        print_info "Backing up all manifests..."
        kubectl get all -n $NAMESPACE -o yaml > "$BACKUP_DIR/all-resources.yaml"
        kubectl get configmaps -n $NAMESPACE -o yaml > "$BACKUP_DIR/configmaps.yaml"
        kubectl get secrets -n $NAMESPACE -o yaml > "$BACKUP_DIR/secrets.yaml"
        kubectl get pvc -n $NAMESPACE -o yaml > "$BACKUP_DIR/pvcs.yaml"

        print_success "Backup saved to: $BACKUP_DIR"
        ;;

    help|--help|-h)
        cat << EOF

${CYAN}Archivist Kubernetes Management${NC}

${GREEN}Usage:${NC}
  ./scripts/k8s-manage.sh [command] [options]

${GREEN}Commands:${NC}
  ${CYAN}status${NC}              Show status of all resources
  ${CYAN}scale${NC} <deploy> <N>  Scale deployment to N replicas
  ${CYAN}logs${NC} <pod>           View logs for pod/deployment
  ${CYAN}exec${NC} <pod>           Execute shell in pod
  ${CYAN}port-forward${NC} <svc>   Forward service port to localhost
  ${CYAN}restart${NC} <deploy>     Restart deployment
  ${CYAN}hpa${NC}                  Show HPA status
  ${CYAN}top${NC}                  Show resource usage
  ${CYAN}update${NC}               Apply latest manifests
  ${CYAN}backup${NC}               Backup all resources
  ${CYAN}delete${NC}               Delete all resources (destructive!)
  ${CYAN}help${NC}                 Show this help

${GREEN}Examples:${NC}
  # Check status
  ./scripts/k8s-manage.sh status

  # Scale workers to 10 replicas
  ./scripts/k8s-manage.sh scale archivist-worker 10

  # View worker logs
  ./scripts/k8s-manage.sh logs archivist-worker

  # Access Neo4j browser
  ./scripts/k8s-manage.sh port-forward neo4j

  # Check autoscaling
  ./scripts/k8s-manage.sh hpa

  # Monitor resource usage
  ./scripts/k8s-manage.sh top

${GREEN}Port Forwarding Services:${NC}
  neo4j, qdrant, redis, kafka, search, graph, worker

EOF
        ;;

    *)
        print_error "Unknown command: $COMMAND"
        print_info "Run './scripts/k8s-manage.sh help' for usage"
        exit 1
        ;;
esac
