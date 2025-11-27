#!/bin/bash

# Recreate Kind Cluster with Proper Best Practices
# This script tears down the existing cluster and creates a new one
# with worker nodes properly configured for storage

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
BLUE='\033[0;34m'
CYAN='\033[0;36m'
NC='\033[0m'

echo -e "${CYAN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║                                                               ║${NC}"
echo -e "${CYAN}║         Kind Cluster Recreation - Best Practices             ║${NC}"
echo -e "${CYAN}║                                                               ║${NC}"
echo -e "${CYAN}╚═══════════════════════════════════════════════════════════════╝${NC}\n"

echo -e "${YELLOW}⚠️  This will DELETE your current 'tkb' cluster and recreate it${NC}"
echo -e "${YELLOW}   with proper worker node configuration.${NC}\n"

echo -e "${BLUE}What this fixes:${NC}"
echo -e "  ✓ Moves workloads OFF control-plane (best practice)"
echo -e "  ✓ Keeps control-plane tainted (production-ready)"
echo -e "  ✓ Mounts storage on worker nodes only"
echo -e "  ✓ Configures proper node affinity\n"

read -p "Continue? [y/N]: " confirm
if [[ ! "$confirm" =~ ^[Yy]$ ]]; then
    echo -e "\n${RED}Cancelled${NC}"
    exit 0
fi

# Step 1: Delete existing cluster
echo -e "\n${BLUE}[1/4]${NC} Deleting existing cluster 'tkb'..."
if kind get clusters | grep -q "^tkb$"; then
    kind delete cluster --name tkb
    echo -e "${GREEN}✓${NC} Cluster deleted\n"
else
    echo -e "${YELLOW}⚠${NC} Cluster 'tkb' not found (maybe already deleted)\n"
fi

# Step 2: Ensure data directory exists
echo -e "${BLUE}[2/4]${NC} Setting up host data directory..."
DATA_DIR="$HOME/archivist-data"
mkdir -p "$DATA_DIR"
echo -e "${GREEN}✓${NC} Created: $DATA_DIR\n"

# Step 3: Create new cluster with proper config
echo -e "${BLUE}[3/4]${NC} Creating new Kind cluster with worker mounts..."

if [ ! -f "kind-config.yaml" ]; then
    echo -e "${RED}✗${NC} kind-config.yaml not found!"
    echo "Please ensure you're running this from the Archivist root directory"
    exit 1
fi

kind create cluster --config kind-config.yaml

echo -e "${GREEN}✓${NC} Cluster created\n"

# Step 4: Verify control-plane taint is present
echo -e "${BLUE}[4/4]${NC} Verifying control-plane taint..."
if kubectl get nodes tkb-control-plane -o jsonpath='{.spec.taints[?(@.key=="node-role.kubernetes.io/control-plane")]}' | grep -q "NoSchedule"; then
    echo -e "${GREEN}✓${NC} Control-plane is properly tainted (workloads will NOT schedule here)\n"
else
    echo -e "${YELLOW}⚠${NC} Control-plane taint not found - adding it...\n"
    kubectl taint nodes tkb-control-plane node-role.kubernetes.io/control-plane:NoSchedule
fi

# Summary
echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                                                               ║${NC}"
echo -e "${GREEN}║                  ✅ Cluster Ready!                             ║${NC}"
echo -e "${GREEN}║                                                               ║${NC}"
echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════╝${NC}\n"

echo -e "${CYAN}📊 Cluster Architecture:${NC}"
kubectl get nodes -o wide

echo -e "\n${CYAN}🔒 Taints (control-plane should be tainted):${NC}"
kubectl get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints

echo -e "\n${CYAN}💾 Storage Configuration:${NC}"
echo -e "  Host:      $DATA_DIR"
echo -e "  Container: /data (mounted on workers only)"

echo -e "\n${CYAN}🎯 Next Step:${NC}"
echo -e "  Run the deployment script:"
echo -e "  ${YELLOW}./scripts/k8s-local-setup.sh${NC}\n"

echo -e "${GREEN}✓ Best practice cluster ready!${NC}"
