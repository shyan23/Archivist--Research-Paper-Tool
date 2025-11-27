#!/bin/bash

# Local Kubernetes Setup for Archivist
# Optimized for offline, personal use on Minikube, Kind, or Docker Desktop

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

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

# Banner
cat << "EOF"

╔═══════════════════════════════════════════════════════════════╗
║                                                               ║
║        Archivist Local Kubernetes Setup                      ║
║        Offline, Personal Use - Autoscaling Enabled           ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝

EOF

NAMESPACE="archivist"

# Check prerequisites
print_header "Checking Prerequisites"

# Check kubectl
if ! command -v kubectl &> /dev/null; then
    print_error "kubectl not found. Please install kubectl"
    print_info "macOS: brew install kubectl"
    print_info "Linux: https://kubernetes.io/docs/tasks/tools/"
    exit 1
fi
print_success "kubectl installed"

# Check cluster connection
if ! kubectl cluster-info &> /dev/null; then
    print_error "Cannot connect to Kubernetes cluster"
    print_info ""
    print_info "Please start one of the following:"
    print_info "  • Docker Desktop: Enable Kubernetes in Settings"
    print_info "  • Minikube: minikube start --cpus=4 --memory=8192"
    print_info "  • Kind: kind create cluster --name archivist"
    exit 1
fi
print_success "Connected to Kubernetes cluster"

# Get cluster info
CLUSTER_NAME=$(kubectl config current-context)
FORCE_RECREATE_CLUSTER=false

# Detect cluster type
if [[ "$CLUSTER_NAME" == *"minikube"* ]]; then
    CLUSTER_TYPE="minikube"
    NODE_NAME=$(kubectl get nodes -o jsonpath='{.items[0].metadata.name}')
elif [[ "$CLUSTER_NAME" == *"kind"* ]]; then
    CLUSTER_TYPE="kind"
    # For Kind, select a WORKER node (not control-plane)
    # Simple approach: get first node that's not control-plane
    NODE_NAME=$(kubectl get nodes -o name | grep -v control-plane | head -1 | cut -d'/' -f2)
    if [ -z "$NODE_NAME" ]; then
        print_error "No worker nodes found in Kind cluster"
        print_info "Please recreate cluster with: kind create cluster --config kind-config.yaml"
        exit 1
    fi
elif [[ "$CLUSTER_NAME" == *"docker-desktop"* ]]; then
    CLUSTER_TYPE="docker-desktop"
    NODE_NAME=$(kubectl get nodes -o jsonpath='{.items[0].metadata.name}')
else
    CLUSTER_TYPE="unknown"
    NODE_NAME=$(kubectl get nodes -o jsonpath='{.items[0].metadata.name}')
fi

print_info "Cluster: ${CYAN}$CLUSTER_NAME${NC}"
print_info "Target node: ${CYAN}$NODE_NAME${NC}"
print_info "Cluster type: ${CYAN}$CLUSTER_TYPE${NC}"

# Validate Kind cluster configuration for best practices
if [[ "$CLUSTER_TYPE" == "kind" ]]; then
    print_header "Validating Cluster Configuration"

    # Check if worker node has /data mount (best practice setup)
    WORKER_HAS_DATA_MOUNT=$(docker exec "$NODE_NAME" ls /data 2>/dev/null && echo "yes" || echo "no")

    # Check if control-plane is tainted
    CONTROL_PLANE_TAINTED=$(kubectl get nodes tkb-control-plane -o jsonpath='{.spec.taints[?(@.key=="node-role.kubernetes.io/control-plane")].effect}' 2>/dev/null)

    if [[ "$WORKER_HAS_DATA_MOUNT" == "no" ]]; then
        print_warning "Worker node is missing /data mount (required for best practices)"
        print_warning "Current cluster was not created with proper kind-config.yaml"
        echo ""
        print_info "For best practices, the cluster should be recreated with:"
        print_info "  • Worker nodes with /data mount from host"
        print_info "  • Control-plane tainted (no workloads)"
        print_info "  • Storage pinned to worker nodes"
        echo ""
        read -p "Do you want to RECREATE the cluster now with best practices? [y/N]: " recreate
        if [[ "$recreate" =~ ^[Yy]$ ]]; then
            FORCE_RECREATE_CLUSTER=true

            # Check if kind-config.yaml exists
            if [ ! -f "kind-config.yaml" ]; then
                print_error "kind-config.yaml not found!"
                print_info "Please run from the Archivist root directory"
                exit 1
            fi

            # Recreate cluster
            print_info "Deleting current cluster..."
            kind delete cluster --name tkb

            print_info "Creating new cluster with best practices..."
            kind create cluster --config kind-config.yaml

            # Re-detect nodes after recreation (get first worker)
            NODE_NAME=$(kubectl get nodes -o name | grep -v control-plane | head -1 | cut -d'/' -f2)

            print_success "Cluster recreated with best practices!"
            echo ""
        else
            print_warning "Continuing with current cluster setup (not ideal for production)"
            print_info "Storage will be configured on control-plane (quick-fix mode)"
            # Override node selection to use control-plane
            NODE_NAME="tkb-control-plane"
            echo ""
        fi
    else
        print_success "Worker node has /data mount configured"
    fi

    if [[ -z "$CONTROL_PLANE_TAINTED" ]] && [[ "$FORCE_RECREATE_CLUSTER" == "false" ]]; then
        print_warning "Control-plane is not tainted (workloads can schedule here)"
        print_info "This works but violates Kubernetes best practices"
    elif [[ -n "$CONTROL_PLANE_TAINTED" ]]; then
        print_success "Control-plane is properly tainted (best practice)"
    fi

    echo ""
fi

# Check metrics server for autoscaling
print_info "Checking metrics server (required for autoscaling)..."
if ! kubectl top nodes &> /dev/null; then
    print_warning "Metrics server not found"

    if [[ "$CLUSTER_TYPE" == "minikube" ]]; then
        print_info "Enabling metrics server for Minikube..."
        minikube addons enable metrics-server
        print_success "Metrics server enabled"
    else
        print_info "Installing metrics server..."
        kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

        # For local clusters, we need to disable TLS verification
        kubectl patch deployment metrics-server -n kube-system --type='json' \
          -p='[{"op": "add", "path": "/spec/template/spec/containers/0/args/-", "value": "--kubelet-insecure-tls"}]' || true

        print_success "Metrics server installed"
    fi

    # Wait for metrics server
    print_info "Waiting for metrics server to be ready..."
    sleep 10
fi
print_success "Metrics server is ready"

# Step 1: Setup data directories
print_header "Setting Up Data Directories"

# For Kind, data is mounted at /data inside the worker container
# For other clusters, we try /data first, then fall back to $HOME
if [[ "$CLUSTER_TYPE" == "kind" ]]; then
    HOST_DATA_DIR="$HOME/archivist-data"

    # Check if we're using best practice setup (worker with /data mount) or quick-fix (control-plane)
    if [[ "$NODE_NAME" == *"worker"* ]]; then
        PV_DATA_DIR="/data"  # Path inside Kind worker container (best practice)
        print_success "Using best practice storage configuration"
        print_info "Host: $HOST_DATA_DIR → Container: $PV_DATA_DIR (on worker)"
    else
        # Quick-fix mode: using control-plane (not ideal but works)
        PV_DATA_DIR="$HOST_DATA_DIR"
        print_warning "Using quick-fix storage configuration (control-plane)"
        print_info "Storage path: $PV_DATA_DIR"
    fi
elif [ -w "/data" ]; then
    HOST_DATA_DIR="/data/archivist"
    PV_DATA_DIR="/data/archivist"
    print_success "Using /data/archivist for storage"
else
    HOST_DATA_DIR="$HOME/archivist-data"
    PV_DATA_DIR="$HOME/archivist-data"
    print_warning "/data is not writable"
    print_info "Using $HOST_DATA_DIR instead"
fi

# Create directories on host
mkdir -p "$HOST_DATA_DIR"/{neo4j,qdrant,redis,kafka,shared}/{lib,tex_files,reports}
print_success "Data directories created at: $HOST_DATA_DIR"

# Step 2: Update PVs with correct node name and paths
print_header "Configuring Persistent Volumes"

PV_FILE="k8s/base/04-local-pvs.yaml"
PV_TEMP=$(mktemp)

# Replace node name and paths in PV file
sed "s/minikube/$NODE_NAME/g" "$PV_FILE" > "$PV_TEMP"

# Replace paths with correct location
sed -i.bak "s|/data/archivist|$PV_DATA_DIR|g" "$PV_TEMP"

print_success "PVs configured for node: $NODE_NAME"
print_success "PV storage path: $PV_DATA_DIR"

# Step 3: Create/Clean namespace
print_header "Preparing Namespace"

# Create namespace if it doesn't exist
kubectl apply -f k8s/base/00-namespace.yaml
print_success "Namespace ready"

# Check if there are existing resources
EXISTING_PODS=$(kubectl get pods -n $NAMESPACE --no-headers 2>/dev/null | wc -l)

if [ "$EXISTING_PODS" -gt 0 ]; then
    print_warning "Found existing deployment in namespace '$NAMESPACE'"
    echo ""
    kubectl get all -n $NAMESPACE
    echo ""
    print_info "🧹 Cleaning up ALL existing resources for fresh deployment..."
    echo ""

    # Step 1: Delete all application resources
    print_info "[1/5] Deleting HPAs..."
    kubectl delete hpa -n $NAMESPACE --all --timeout=30s 2>/dev/null || true

    print_info "[2/5] Deleting deployments and statefulsets..."
    kubectl delete deployments,statefulsets -n $NAMESPACE --all --timeout=60s 2>/dev/null || true

    print_info "[3/5] Deleting services..."
    kubectl delete services -n $NAMESPACE --all --timeout=30s 2>/dev/null || true

    print_info "[4/5] Deleting PVCs and PVs..."
    kubectl delete pvc -n $NAMESPACE --all --timeout=60s 2>/dev/null || true
    # Delete PVs that belong to this deployment
    kubectl delete pv neo4j-pv qdrant-pv redis-pv kafka-pv archivist-data-pv 2>/dev/null || true

    print_info "[5/5] Waiting for all pods to terminate..."
    kubectl wait --for=delete pod --all -n $NAMESPACE --timeout=90s 2>/dev/null || true

    print_info "Deleting any remaining configmaps and secrets (except archivist-secrets)..."
    kubectl delete configmap -n $NAMESPACE --all 2>/dev/null || true

    # Short wait to ensure everything is cleaned up
    sleep 3

    print_success "✅ Complete cleanup finished - ready for fresh deployment"
    echo ""
else
    print_info "No existing resources found - proceeding with fresh deployment"
fi

# Step 4: Configure secrets
print_header "Configuring Secrets"

# Check if secrets exist
if kubectl get secret archivist-secrets -n $NAMESPACE &> /dev/null; then
    print_warning "Secrets already exist"
    read -p "Do you want to update them? [y/N]: " update_secrets
    if [[ "$update_secrets" =~ ^[Yy]$ ]]; then
        kubectl delete secret archivist-secrets -n $NAMESPACE
    else
        print_info "Keeping existing secrets"
    fi
fi

if ! kubectl get secret archivist-secrets -n $NAMESPACE &> /dev/null; then
    # Check for .env file
    if [ -f ".env" ]; then
        GEMINI_API_KEY=$(grep GEMINI_API_KEY .env | cut -d '=' -f2 | tr -d ' ')
    fi

    if [ -z "$GEMINI_API_KEY" ] || [ "$GEMINI_API_KEY" == "your-gemini-api-key-here" ]; then
        read -p "Enter your Gemini API key: " GEMINI_API_KEY
    fi

    read -p "Enter Neo4j password (default: password): " NEO4J_PASSWORD
    NEO4J_PASSWORD=${NEO4J_PASSWORD:-password}

    kubectl create secret generic archivist-secrets \
        --from-literal=GEMINI_API_KEY="$GEMINI_API_KEY" \
        --from-literal=NEO4J_PASSWORD="$NEO4J_PASSWORD" \
        -n $NAMESPACE

    print_success "Secrets created"
fi

# Step 5: Deploy ConfigMaps
print_header "Deploying Configuration"
kubectl apply -f k8s/base/01-configmap.yaml
print_success "ConfigMaps applied"

# Step 6: Deploy Storage
print_header "Deploying Storage"
kubectl apply -f "$PV_TEMP"
kubectl apply -f k8s/base/03-pvcs.yaml
print_success "Persistent Volumes and Claims created"

print_info "Waiting for PVCs to be bound (max 60s)..."
kubectl wait --for=jsonpath='{.status.phase}'=Bound pvc --all -n $NAMESPACE --timeout=60s || print_warning "Some PVCs may need more time"

# Step 7: Deploy StatefulSets
print_header "Deploying Stateful Services"
kubectl apply -f k8s/base/10-neo4j-statefulset.yaml
print_success "Neo4j deployed"

kubectl apply -f k8s/base/11-kafka-statefulset.yaml
print_success "Kafka deployed"

print_info "Waiting for StatefulSets to be ready (this may take 2-3 minutes)..."
kubectl wait --for=condition=ready pod -l app=neo4j -n $NAMESPACE --timeout=300s &
kubectl wait --for=condition=ready pod -l app=kafka -n $NAMESPACE --timeout=300s &
wait

# Step 8: Deploy Services
print_header "Deploying Application Services"
kubectl apply -f k8s/base/20-qdrant-deployment.yaml
print_success "Qdrant deployed"

kubectl apply -f k8s/base/21-redis-deployment.yaml
print_success "Redis deployed"

kubectl apply -f k8s/base/30-archivist-worker-deployment.yaml
print_success "Archivist workers deployed"

kubectl apply -f k8s/base/31-search-service-deployment.yaml
print_success "Search service deployed"

kubectl apply -f k8s/base/32-graph-service-deployment.yaml
print_success "Graph service deployed"

print_info "Waiting for deployments to be ready..."
sleep 10

# Step 9: Deploy Autoscaling
print_header "Configuring Autoscaling"
kubectl apply -f k8s/base/40-hpa.yaml
print_success "Horizontal Pod Autoscalers configured"

# Clean up temp file
rm -f "$PV_TEMP" "${PV_TEMP}.bak"

# Summary
print_header "✅ Deployment Complete!"

echo -e "${GREEN}Archivist is now running on your local Kubernetes cluster!${NC}\n"

echo -e "${CYAN}📊 Cluster Status:${NC}"
kubectl get pods -n $NAMESPACE

echo -e "\n${CYAN}📈 Autoscaling Status:${NC}"
kubectl get hpa -n $NAMESPACE

echo -e "\n${CYAN}💾 Data Storage:${NC}"
echo -e "  Host Directory: $HOST_DATA_DIR/"
echo -e "  Papers/Reports: $HOST_DATA_DIR/shared/"
echo -e "  Neo4j Data: $HOST_DATA_DIR/neo4j/"
echo -e "  Qdrant Data: $HOST_DATA_DIR/qdrant/"

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
echo -e "4. ${YELLOW}Watch autoscaling in action:${NC}"
echo -e "   watch kubectl get hpa -n $NAMESPACE"
echo ""
echo -e "5. ${YELLOW}Scale manually if needed:${NC}"
echo -e "   kubectl scale deployment/archivist-worker --replicas=4 -n $NAMESPACE"

echo -e "\n${CYAN}📚 Documentation:${NC}"
echo -e "  Full Guide: docs/KUBERNETES_LOCAL_DEPLOYMENT.md"
echo -e "  Management: ./scripts/k8s-manage.sh status"

echo -e "\n${CYAN}🌐 Access URLs (after port-forward):${NC}"
echo -e "  Neo4j:  http://localhost:7474"
echo -e "  Qdrant: http://localhost:6333/dashboard"
echo -e "  Search: http://localhost:8000/docs"

print_success "Setup complete! Happy processing! 🚀"
