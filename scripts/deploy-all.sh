#!/bin/bash

# Archivist All-In-One Deployment Script
# This script does EVERYTHING: cluster setup, deployment, image building, and verification

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
MAGENTA='\033[0;35m'
CYAN='\033[0;36m'
NC='\033[0m'

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

# Banner
cat << "EOF"

╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║              ARCHIVIST ALL-IN-ONE DEPLOYMENT                 ║
║                                                               ║
║    Cluster Setup → Cleanup → Deploy → Build → Verify         ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝

EOF

NAMESPACE="archivist"
CLUSTER_NAME="tkb"

# Check prerequisites
print_header "Step 1/7: Checking Prerequisites"

if ! command -v kubectl &> /dev/null; then
    print_error "kubectl not found"
    exit 1
fi
print_success "kubectl installed"

if ! command -v kind &> /dev/null; then
    print_error "kind not found"
    exit 1
fi
print_success "kind installed"

if ! command -v docker &> /dev/null; then
    print_error "docker not found"
    exit 1
fi
print_success "docker installed"

# Check if cluster exists and validate it
print_header "Step 2/7: Cluster Validation & Setup"

CLUSTER_EXISTS=$(kind get clusters 2>/dev/null | grep -c "^${CLUSTER_NAME}$" || true)

if [ "$CLUSTER_EXISTS" -eq 0 ]; then
    print_info "Cluster '${CLUSTER_NAME}' not found"
    print_info "Creating new cluster with best practices..."

    if [ ! -f "kind-config.yaml" ]; then
        print_error "kind-config.yaml not found!"
        print_info "Please run from the Archivist root directory"
        exit 1
    fi

    kind create cluster --config kind-config.yaml
    print_success "Cluster created"
else
    print_info "Cluster '${CLUSTER_NAME}' exists - validating configuration..."

    # Set context
    kubectl config use-context kind-${CLUSTER_NAME} &>/dev/null

    # Check if workers have /data mount
    WORKER_NODE=$(kubectl get nodes -o name | grep -v control-plane | head -1 | cut -d'/' -f2)

    if [ -z "$WORKER_NODE" ]; then
        print_error "No worker nodes found!"
        exit 1
    fi

    WORKER_HAS_DATA=$(docker exec "$WORKER_NODE" ls /data 2>/dev/null && echo "yes" || echo "no")

    if [[ "$WORKER_HAS_DATA" == "no" ]]; then
        print_warning "Worker node missing /data mount - cluster needs recreation"
        print_info "Recreating cluster with proper configuration..."

        kind delete cluster --name ${CLUSTER_NAME}
        kind create cluster --config kind-config.yaml
        print_success "Cluster recreated with best practices"
    else
        print_success "Cluster is properly configured"
    fi
fi

# Ensure metrics server is installed
print_info "Checking metrics server..."
if ! kubectl top nodes &> /dev/null; then
    print_info "Installing metrics server..."
    kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml &>/dev/null
    kubectl patch deployment metrics-server -n kube-system --type='json' \
      -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--kubelet-insecure-tls"}]' &>/dev/null || true
    sleep 5
fi
print_success "Metrics server ready"

# Get worker node for deployment
WORKER_NODE=$(kubectl get nodes -o name | grep -v control-plane | head -1 | cut -d'/' -f2)
print_info "Target node: ${CYAN}${WORKER_NODE}${NC}"

# Setup data directories
print_header "Step 3/7: Setting Up Data Directories"

HOST_DATA_DIR="$HOME/archivist-data"
PV_DATA_DIR="/data"

mkdir -p "$HOST_DATA_DIR"/{neo4j,qdrant,redis,kafka,shared}/{lib,tex_files,reports}
print_success "Data directories created at: $HOST_DATA_DIR"
print_info "Container path: $PV_DATA_DIR"

# Cleanup existing deployment
print_header "Step 4/7: Cleaning Up Existing Deployment"

if kubectl get namespace $NAMESPACE &>/dev/null; then
    print_info "Found existing namespace - performing complete cleanup..."

    # Delete all resources
    kubectl delete hpa -n $NAMESPACE --all --timeout=30s &>/dev/null || true
    kubectl delete deployments,statefulsets -n $NAMESPACE --all --timeout=60s &>/dev/null || true
    kubectl delete services -n $NAMESPACE --all --timeout=30s &>/dev/null || true
    kubectl delete pvc -n $NAMESPACE --all --timeout=60s &>/dev/null || true
    kubectl delete pv neo4j-pv qdrant-pv redis-pv kafka-pv archivist-data-pv &>/dev/null || true
    kubectl delete configmap -n $NAMESPACE --all &>/dev/null || true
    kubectl delete namespace $NAMESPACE --timeout=60s &>/dev/null || true

    print_success "Cleanup complete"
else
    print_info "No existing deployment found"
fi

# Deploy Kubernetes resources
print_header "Step 5/7: Deploying Kubernetes Resources"

# Create namespace
kubectl create namespace $NAMESPACE
print_success "Namespace created"

# Create secrets
print_info "Creating secrets..."
GEMINI_API_KEY="${GEMINI_API_KEY:-}"
if [ -z "$GEMINI_API_KEY" ] && [ -f ".env" ]; then
    GEMINI_API_KEY=$(grep GEMINI_API_KEY .env 2>/dev/null | cut -d '=' -f2 | tr -d ' ' || echo "")
fi

if [ -z "$GEMINI_API_KEY" ]; then
    print_warning "GEMINI_API_KEY not set in environment or .env file"
    read -p "Enter your Gemini API key: " GEMINI_API_KEY
fi

NEO4J_PASSWORD="${NEO4J_PASSWORD:-password}"

kubectl create secret generic archivist-secrets \
    --from-literal=GEMINI_API_KEY="$GEMINI_API_KEY" \
    --from-literal=NEO4J_PASSWORD="$NEO4J_PASSWORD" \
    -n $NAMESPACE
print_success "Secrets created"

# Create ConfigMap
kubectl apply -f k8s/base/01-configmap.yaml
print_success "ConfigMap applied"

# Create PVs with correct node affinity
print_info "Creating Persistent Volumes on worker node..."
PV_TEMP=$(mktemp)
sed "s/minikube/$WORKER_NODE/g" k8s/base/04-local-pvs.yaml | \
    sed "s|/data/archivist|$PV_DATA_DIR|g" > "$PV_TEMP"

kubectl apply -f "$PV_TEMP"
kubectl apply -f k8s/base/03-pvcs.yaml
rm -f "$PV_TEMP"
print_success "Storage configured"

# Deploy StatefulSets
print_info "Deploying Neo4j and Kafka..."
kubectl apply -f k8s/base/10-neo4j-statefulset.yaml
kubectl apply -f k8s/base/11-kafka-statefulset.yaml
print_success "StatefulSets deployed"

# Deploy Deployments
print_info "Deploying Qdrant, Redis, and services..."
kubectl apply -f k8s/base/20-qdrant-deployment.yaml
kubectl apply -f k8s/base/21-redis-deployment.yaml
kubectl apply -f k8s/base/30-archivist-worker-deployment.yaml
kubectl apply -f k8s/base/31-search-service-deployment.yaml
kubectl apply -f k8s/base/32-graph-service-deployment.yaml
print_success "Deployments created"

# Deploy HPA
kubectl apply -f k8s/base/40-hpa.yaml
print_success "Autoscaling configured"

# Build and load Docker images
print_header "Step 6/7: Building & Loading Docker Images"

print_info "[1/3] Building archivist:latest..."
docker build -t archivist:latest -f Dockerfile . -q
print_success "Built archivist:latest"

print_info "[2/3] Building archivist-graph:latest..."
docker build -t archivist-graph:latest -f services/graph-service/Dockerfile ./services/graph-service -q
print_success "Built archivist-graph:latest"

print_info "[3/3] Building archivist-search:latest..."
docker build -t archivist-search:latest -f services/search-engine/Dockerfile ./services/search-engine -q
print_success "Built archivist-search:latest"

print_info "Loading images into Kind cluster..."
kind load docker-image archivist:latest --name ${CLUSTER_NAME}
kind load docker-image archivist-graph:latest --name ${CLUSTER_NAME}
kind load docker-image archivist-search:latest --name ${CLUSTER_NAME}
print_success "Images loaded into cluster"

# Restart pods to pick up new images
print_info "Restarting pods to use new images..."
kubectl delete pod -n $NAMESPACE -l app=archivist-worker &>/dev/null || true
kubectl delete pod -n $NAMESPACE -l app=graph-service &>/dev/null || true
kubectl delete pod -n $NAMESPACE -l app=search-service &>/dev/null || true
print_success "Pods restarted"

# Wait and verify
print_header "Step 7/7: Verification & Status"

print_info "Waiting for base services to be ready (max 3 minutes)..."
sleep 10

# Check pod status
echo ""
echo -e "${CYAN}📊 Pod Status:${NC}"
kubectl get pods -n $NAMESPACE -o wide

echo ""
echo -e "${CYAN}💾 Storage Status:${NC}"
kubectl get pvc -n $NAMESPACE

echo ""
echo -e "${CYAN}📈 Autoscaling Status:${NC}"
kubectl get hpa -n $NAMESPACE

echo ""
echo -e "${CYAN}🔍 Node Distribution:${NC}"
kubectl get pods -n $NAMESPACE -o custom-columns=NAME:.metadata.name,NODE:.spec.nodeName,STATUS:.status.phase --no-headers | sort -k2

# Final summary
echo ""
print_header "✅ Deployment Complete!"

echo -e "${GREEN}Archivist is deployed with best practices!${NC}\n"

echo -e "${CYAN}📁 Data Storage:${NC}"
echo -e "  Host: $HOST_DATA_DIR"
echo -e "  Container: $PV_DATA_DIR (on worker nodes)"

echo -e "\n${CYAN}🎯 Next Steps:${NC}"
echo ""
echo -e "1. ${YELLOW}Copy papers to process:${NC}"
echo -e "   cp your_papers/*.pdf $HOST_DATA_DIR/shared/lib/"
echo ""
echo -e "2. ${YELLOW}Access services (port forwarding):${NC}"
echo -e "   kubectl port-forward svc/neo4j-service 7474:7474 -n $NAMESPACE"
echo -e "   kubectl port-forward svc/qdrant-service 6333:6333 -n $NAMESPACE"
echo ""
echo -e "3. ${YELLOW}Monitor processing:${NC}"
echo -e "   kubectl logs -f deployment/archivist-worker -n $NAMESPACE"
echo ""
echo -e "4. ${YELLOW}Watch pods:${NC}"
echo -e "   watch kubectl get pods -n $NAMESPACE"

echo ""
print_success "All done! 🚀"
