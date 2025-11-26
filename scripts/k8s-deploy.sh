#!/bin/bash

# Kubernetes Deployment Script for Archivist
# Deploys all services to Kubernetes cluster with autoscaling

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
║        Archivist Kubernetes Deployment                       ║
║        Scalable, Production-Ready Setup                      ║
║                                                               ║
╚═══════════════════════════════════════════════════════════════╝

EOF

# Check prerequisites
print_header "Checking Prerequisites"

# Check kubectl
if ! command -v kubectl &> /dev/null; then
    print_error "kubectl not found. Please install kubectl"
    exit 1
fi
print_success "kubectl installed"

# Check kubectl connection
if ! kubectl cluster-info &> /dev/null; then
    print_error "Cannot connect to Kubernetes cluster"
    print_info "Please configure kubectl to connect to your cluster"
    exit 1
fi
print_success "Connected to Kubernetes cluster"

# Get cluster info
CLUSTER_NAME=$(kubectl config current-context)
print_info "Deploying to cluster: ${CYAN}$CLUSTER_NAME${NC}"

# Check Docker (for building images)
if ! command -v docker &> /dev/null; then
    print_warning "Docker not found. You'll need to build images manually"
else
    print_success "Docker installed"
fi

# Parse command line arguments
BUILD_IMAGES=${1:-"no"}
NAMESPACE="archivist"

# Step 1: Build Docker images
if [ "$BUILD_IMAGES" = "build" ]; then
    print_header "Building Docker Images"

    # Build main application
    print_info "Building Archivist worker image..."
    docker build -t archivist:latest -f Dockerfile.k8s .
    print_success "Archivist worker image built"

    # Build search service
    print_info "Building search service image..."
    docker build -t archivist-search:latest -f services/search-engine/Dockerfile.k8s services/search-engine/
    print_success "Search service image built"

    # Tag images for registry (optional)
    read -p "Do you want to push images to a registry? [y/N]: " push_images
    if [[ "$push_images" =~ ^[Yy]$ ]]; then
        read -p "Enter your Docker registry (e.g., docker.io/username): " registry

        docker tag archivist:latest $registry/archivist:latest
        docker tag archivist-search:latest $registry/archivist-search:latest

        print_info "Pushing images to registry..."
        docker push $registry/archivist:latest
        docker push $registry/archivist-search:latest
        print_success "Images pushed to registry"

        # Update deployment files with registry
        print_info "Update k8s/base/*-deployment.yaml with your registry URL"
    fi
fi

# Step 2: Create namespace
print_header "Creating Namespace"
kubectl apply -f k8s/base/00-namespace.yaml
print_success "Namespace created"

# Step 3: Setup secrets
print_header "Setting Up Secrets"

if kubectl get secret archivist-secrets -n $NAMESPACE &> /dev/null; then
    print_warning "Secrets already exist, skipping..."
else
    print_warning "Creating secrets from template"
    print_info "You need to update the secrets with actual values!"

    # Check if .env file exists
    if [ -f ".env" ]; then
        GEMINI_API_KEY=$(grep GEMINI_API_KEY .env | cut -d '=' -f2)

        if [ -n "$GEMINI_API_KEY" ] && [ "$GEMINI_API_KEY" != "your-gemini-api-key-here" ]; then
            kubectl create secret generic archivist-secrets \
                --from-literal=GEMINI_API_KEY="$GEMINI_API_KEY" \
                --from-literal=NEO4J_PASSWORD="password" \
                -n $NAMESPACE
            print_success "Secrets created from .env file"
        else
            print_warning "No valid GEMINI_API_KEY in .env file"
            kubectl apply -f k8s/base/02-secrets.yaml
            print_error "Please update secrets manually:"
            print_info "kubectl create secret generic archivist-secrets \\"
            print_info "  --from-literal=GEMINI_API_KEY='your-key' \\"
            print_info "  --from-literal=NEO4J_PASSWORD='your-password' \\"
            print_info "  -n $NAMESPACE --dry-run=client -o yaml | kubectl apply -f -"
        fi
    else
        kubectl apply -f k8s/base/02-secrets.yaml
        print_warning "Update secrets with actual values"
    fi
fi

# Step 4: Apply ConfigMaps
print_header "Applying ConfigMaps"
kubectl apply -f k8s/base/01-configmap.yaml
print_success "ConfigMaps applied"

# Step 5: Create PVCs
print_header "Creating Persistent Volume Claims"
kubectl apply -f k8s/base/03-pvcs.yaml
print_success "PVCs created"

print_info "Waiting for PVCs to be bound..."
kubectl wait --for=jsonpath='{.status.phase}'=Bound pvc --all -n $NAMESPACE --timeout=300s || true

# Step 6: Deploy StatefulSets
print_header "Deploying StatefulSets (Neo4j, Kafka)"
kubectl apply -f k8s/base/10-neo4j-statefulset.yaml
kubectl apply -f k8s/base/11-kafka-statefulset.yaml
print_success "StatefulSets deployed"

print_info "Waiting for StatefulSets to be ready..."
kubectl wait --for=condition=ready pod -l app=neo4j -n $NAMESPACE --timeout=300s || print_warning "Neo4j may need more time"
kubectl wait --for=condition=ready pod -l app=kafka -n $NAMESPACE --timeout=300s || print_warning "Kafka may need more time"

# Step 7: Deploy standard services
print_header "Deploying Services (Qdrant, Redis)"
kubectl apply -f k8s/base/20-qdrant-deployment.yaml
kubectl apply -f k8s/base/21-redis-deployment.yaml
print_success "Services deployed"

print_info "Waiting for services to be ready..."
kubectl wait --for=condition=available deployment/qdrant -n $NAMESPACE --timeout=300s || print_warning "Qdrant may need more time"
kubectl wait --for=condition=available deployment/redis -n $NAMESPACE --timeout=300s || print_warning "Redis may need more time"

# Step 8: Deploy application workers
print_header "Deploying Application Services"
kubectl apply -f k8s/base/30-archivist-worker-deployment.yaml
kubectl apply -f k8s/base/31-search-service-deployment.yaml
kubectl apply -f k8s/base/32-graph-service-deployment.yaml
print_success "Application services deployed"

# Step 9: Deploy autoscaling
print_header "Configuring Autoscaling (HPA)"
kubectl apply -f k8s/base/40-hpa.yaml
print_success "Horizontal Pod Autoscalers configured"

# Step 10: Deploy ingress (optional)
read -p "Do you want to deploy Ingress? [y/N]: " deploy_ingress
if [[ "$deploy_ingress" =~ ^[Yy]$ ]]; then
    print_header "Deploying Ingress"
    print_warning "Make sure to update the host in k8s/base/50-ingress.yaml"
    kubectl apply -f k8s/base/50-ingress.yaml
    print_success "Ingress deployed"
fi

# Summary
print_header "Deployment Summary"

echo -e "${GREEN}Deployment complete!${NC}\n"

echo -e "${CYAN}View all resources:${NC}"
echo "  kubectl get all -n $NAMESPACE"
echo ""

echo -e "${CYAN}Check pod status:${NC}"
echo "  kubectl get pods -n $NAMESPACE"
echo ""

echo -e "${CYAN}View logs:${NC}"
echo "  kubectl logs -f deployment/archivist-worker -n $NAMESPACE"
echo ""

echo -e "${CYAN}Check HPA status:${NC}"
echo "  kubectl get hpa -n $NAMESPACE"
echo ""

echo -e "${CYAN}Access services (port-forward):${NC}"
echo "  Neo4j:  kubectl port-forward svc/neo4j-service 7474:7474 -n $NAMESPACE"
echo "  Qdrant: kubectl port-forward svc/qdrant-service 6333:6333 -n $NAMESPACE"
echo "  Redis:  kubectl port-forward svc/redis-service 6379:6379 -n $NAMESPACE"
echo ""

echo -e "${CYAN}Scale manually:${NC}"
echo "  kubectl scale deployment/archivist-worker --replicas=10 -n $NAMESPACE"
echo ""

echo -e "${CYAN}Monitor autoscaling:${NC}"
echo "  watch kubectl get hpa -n $NAMESPACE"
echo ""

print_success "All services deployed and ready!"
