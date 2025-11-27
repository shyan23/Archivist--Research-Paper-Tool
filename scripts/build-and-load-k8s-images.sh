#!/bin/bash

# Build and load Docker images for Kind cluster
# This script builds the three custom images needed for Kubernetes deployment

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

echo -e "${BLUE}════════════════════════════════════════${NC}"
echo -e "${BLUE}  Building Archivist K8s Docker Images  ${NC}"
echo -e "${BLUE}════════════════════════════════════════${NC}\n"

# Detect Kind cluster name
CLUSTER_NAME=$(kubectl config current-context | sed 's/kind-//')

if [[ ! "$CLUSTER_NAME" ]]; then
    echo -e "${YELLOW}⚠️  Could not detect Kind cluster${NC}"
    echo "Please run: kind create cluster --name <name>"
    exit 1
fi

echo -e "${GREEN}✓${NC} Detected Kind cluster: ${BLUE}$CLUSTER_NAME${NC}\n"

# Build image 1: Main Archivist Worker
echo -e "${BLUE}[1/3]${NC} Building archivist:latest..."
docker build -t archivist:latest -f Dockerfile . || {
    echo -e "${YELLOW}⚠️  Failed to build archivist:latest${NC}"
    exit 1
}
echo -e "${GREEN}✓${NC} Built archivist:latest\n"

# Build image 2: Graph Service
echo -e "${BLUE}[2/3]${NC} Building archivist-graph:latest..."
docker build -t archivist-graph:latest -f services/graph-service/Dockerfile ./services/graph-service || {
    echo -e "${YELLOW}⚠️  Failed to build archivist-graph:latest${NC}"
    exit 1
}
echo -e "${GREEN}✓${NC} Built archivist-graph:latest\n"

# Build image 3: Search Service
echo -e "${BLUE}[3/3]${NC} Building archivist-search:latest..."
docker build -t archivist-search:latest -f services/search-engine/Dockerfile ./services/search-engine || {
    echo -e "${YELLOW}⚠️  Failed to build archivist-search:latest${NC}"
    exit 1
}
echo -e "${GREEN}✓${NC} Built archivist-search:latest\n"

echo -e "${BLUE}════════════════════════════════════════${NC}"
echo -e "${BLUE}  Loading Images into Kind Cluster      ${NC}"
echo -e "${BLUE}════════════════════════════════════════${NC}\n"

# Load images into Kind cluster
echo -e "${BLUE}[1/3]${NC} Loading archivist:latest..."
kind load docker-image archivist:latest --name "$CLUSTER_NAME"
echo -e "${GREEN}✓${NC} Loaded archivist:latest\n"

echo -e "${BLUE}[2/3]${NC} Loading archivist-graph:latest..."
kind load docker-image archivist-graph:latest --name "$CLUSTER_NAME"
echo -e "${GREEN}✓${NC} Loaded archivist-graph:latest\n"

echo -e "${BLUE}[3/3]${NC} Loading archivist-search:latest..."
kind load docker-image archivist-search:latest --name "$CLUSTER_NAME"
echo -e "${GREEN}✓${NC} Loaded archivist-search:latest\n"

echo -e "${GREEN}════════════════════════════════════════${NC}"
echo -e "${GREEN}  ✅ All images built and loaded!       ${NC}"
echo -e "${GREEN}════════════════════════════════════════${NC}\n"

echo "Restarting failed pods to pick up new images..."
kubectl delete pod -n archivist -l app=graph-service --ignore-not-found=true
kubectl delete pod -n archivist -l app=search-service --ignore-not-found=true
kubectl delete pod -n archivist -l app=archivist-worker --ignore-not-found=true

echo -e "\n${GREEN}✓${NC} Done! Check pod status with:"
echo "   kubectl get pods -n archivist"
