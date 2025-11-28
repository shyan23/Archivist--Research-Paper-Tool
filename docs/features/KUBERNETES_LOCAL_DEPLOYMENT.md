# Local/Offline Kubernetes Deployment Guide

**Deploy Archivist on your personal machine with Kubernetes for scalable, offline paper processing.**

---

## 🎯 Why Kubernetes for Local/Offline Use?

While Archivist works great standalone, Kubernetes provides benefits even for offline, personal use:

✅ **Autoscaling** - Automatically add/remove workers based on load
✅ **Resource Management** - Better CPU/memory utilization
✅ **Service Isolation** - Each component runs independently
✅ **Easy Restart** - Automatic recovery from crashes
✅ **Consistent Environment** - Same setup across machines
✅ **Learning** - Great way to learn Kubernetes

**This is NOT for cloud deployment** - it's optimized for running on your laptop/desktop completely offline.

---

## 📋 Prerequisites

### System Requirements

**Minimum:**
- **CPU**: 4 cores
- **RAM**: 8GB
- **Disk**: 50GB free space
- **OS**: Linux, macOS, or Windows with WSL2

**Recommended:**
- **CPU**: 8+ cores
- **RAM**: 16GB+
- **Disk**: 100GB+ free space

### Required Software

```bash
# 1. Docker Desktop (includes Kubernetes)
# Download from: https://www.docker.com/products/docker-desktop

# OR Minikube
# macOS
brew install minikube

# Linux
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube

# 2. kubectl
# macOS
brew install kubectl

# Linux
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl
```

---

## 🚀 Quick Start

### Option 1: Docker Desktop (Easiest)

```bash
# 1. Enable Kubernetes in Docker Desktop
# Settings → Kubernetes → Enable Kubernetes → Apply

# 2. Verify Kubernetes is running
kubectl cluster-info

# 3. Deploy Archivist
cd Archivist
./scripts/k8s-local-setup.sh
```

### Option 2: Minikube

```bash
# 1. Start Minikube with sufficient resources
minikube start \
  --cpus=4 \
  --memory=8192 \
  --disk-size=50g \
  --driver=docker

# 2. Enable metrics server (for autoscaling)
minikube addons enable metrics-server

# 3. Deploy Archivist
cd Archivist
./scripts/k8s-local-setup.sh
```

### Option 3: Kind (Kubernetes in Docker)

```bash
# 1. Install Kind
# macOS
brew install kind

# Linux
curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
chmod +x ./kind
sudo mv ./kind /usr/local/bin/kind

# 2. Create cluster
kind create cluster --name archivist --config k8s/kind-config.yaml

# 3. Deploy Archivist
cd Archivist
./scripts/k8s-local-setup.sh
```

---

## 📦 What Gets Deployed

```
┌─────────────────────────────────────────┐
│         Your Local Machine              │
│                                         │
│  ┌───────────────────────────────────┐ │
│  │    Local Kubernetes Cluster       │ │
│  │                                   │ │
│  │  • Archivist Workers (1-4 pods)  │ │
│  │  • Neo4j (1 pod)                 │ │
│  │  • Qdrant (1-2 pods)             │ │
│  │  • Redis (1 pod)                 │ │
│  │  • Kafka (1 pod)                 │ │
│  │  • Search Service (1-2 pods)     │ │
│  │  • Graph Service (1-2 pods)      │ │
│  └───────────────────────────────────┘ │
│                                         │
│  Data Storage: /data/archivist/         │
│  • neo4j/     (graph data)              │
│  • qdrant/    (vector embeddings)       │
│  • redis/     (cache)                   │
│  • kafka/     (message queue)           │
│  • shared/    (papers, reports)         │
└─────────────────────────────────────────┘
```

### Autoscaling Limits (Offline-Optimized)

| Service | Min Pods | Max Pods | Trigger |
|---------|----------|----------|---------|
| **Workers** | 1 | 4 | CPU > 75% |
| **Qdrant** | 1 | 2 | CPU > 80% |
| **Search** | 1 | 2 | CPU > 80% |
| **Graph** | 1 | 2 | CPU > 80% |

**Total Resource Usage:**
- Idle: ~2-3GB RAM, 1-2 CPU cores
- Full Load: ~6-8GB RAM, 4-8 CPU cores

---

## 🔧 Setup Steps (Detailed)

### Step 1: Prepare Data Directories

```bash
# Create local directories for persistent storage
sudo mkdir -p /data/archivist/{neo4j,qdrant,redis,kafka,shared}
sudo chown -R $USER:$USER /data/archivist

# Or use home directory (no sudo needed)
mkdir -p $HOME/archivist-data/{neo4j,qdrant,redis,kafka,shared}
```

### Step 2: Update Local PVs

Edit `k8s/base/04-local-pvs.yaml` and update:
1. Node name (if not using minikube)
2. Storage paths

```bash
# Get your node name
kubectl get nodes

# Update the yaml file
# Change "minikube" to your node name (or "docker-desktop")
# Change paths to your actual paths
```

### Step 3: Create Secrets

```bash
# Create namespace first
kubectl apply -f k8s/base/00-namespace.yaml

# Create secrets
kubectl create secret generic archivist-secrets \
  --from-literal=GEMINI_API_KEY='your-api-key-here' \
  --from-literal=NEO4J_PASSWORD='your-password' \
  -n archivist
```

### Step 4: Deploy All Components

```bash
# Apply all manifests in order
kubectl apply -f k8s/base/01-configmap.yaml
kubectl apply -f k8s/base/03-pvcs.yaml
kubectl apply -f k8s/base/04-local-pvs.yaml
kubectl apply -f k8s/base/10-neo4j-statefulset.yaml
kubectl apply -f k8s/base/11-kafka-statefulset.yaml
kubectl apply -f k8s/base/20-qdrant-deployment.yaml
kubectl apply -f k8s/base/21-redis-deployment.yaml
kubectl apply -f k8s/base/30-archivist-worker-deployment.yaml
kubectl apply -f k8s/base/31-search-service-deployment.yaml
kubectl apply -f k8s/base/32-graph-service-deployment.yaml
kubectl apply -f k8s/base/40-hpa.yaml
```

### Step 5: Verify Deployment

```bash
# Check all pods are running
kubectl get pods -n archivist

# Expected output:
# NAME                                READY   STATUS    RESTARTS
# archivist-worker-xxx                1/1     Running   0
# neo4j-0                             1/1     Running   0
# qdrant-xxx                          1/1     Running   0
# redis-xxx                           1/1     Running   0
# kafka-0                             1/1     Running   0
# search-service-xxx                  1/1     Running   0
# graph-service-xxx                   1/1     Running   0

# Check autoscaling status
kubectl get hpa -n archivist
```

---

## 🎮 Usage

### Access Services Locally

**Port Forwarding:**

```bash
# Neo4j Browser
kubectl port-forward svc/neo4j-service 7474:7474 7687:7687 -n archivist &
# Open: http://localhost:7474

# Qdrant Dashboard
kubectl port-forward svc/qdrant-service 6333:6333 -n archivist &
# Open: http://localhost:6333/dashboard

# Redis
kubectl port-forward svc/redis-service 6379:6379 -n archivist &

# Search API
kubectl port-forward svc/search-service 8000:8000 -n archivist &
# Open: http://localhost:8000/docs
```

### Process Papers

```bash
# Copy papers to shared storage
cp your_papers/*.pdf /data/archivist/shared/lib/

# Or if using home directory
cp your_papers/*.pdf $HOME/archivist-data/shared/lib/

# Watch workers process them
kubectl logs -f deployment/archivist-worker -n archivist

# Monitor autoscaling
watch kubectl get hpa -n archivist
```

### Scale Manually

```bash
# Scale workers to 4 (max for local)
kubectl scale deployment/archivist-worker --replicas=4 -n archivist

# Scale back down
kubectl scale deployment/archivist-worker --replicas=1 -n archivist
```

### View Logs

```bash
# Worker logs
kubectl logs -f deployment/archivist-worker -n archivist

# Neo4j logs
kubectl logs -f neo4j-0 -n archivist

# All logs from a service
kubectl logs -f deployment/search-service -n archivist --all-containers
```

---

## 📊 Monitoring

### Resource Usage

```bash
# Pod resource usage
kubectl top pods -n archivist

# Node resource usage
kubectl top nodes

# Watch in real-time
watch kubectl top pods -n archivist
```

### Dashboard (Optional)

```bash
# For Minikube
minikube dashboard

# For Kind/Docker Desktop, install Kubernetes Dashboard
kubectl apply -f https://raw.githubusercontent.com/kubernetes/dashboard/v2.7.0/aio/deploy/recommended.yaml

# Create admin user
kubectl create serviceaccount dashboard-admin -n kubernetes-dashboard
kubectl create clusterrolebinding dashboard-admin \
  --clusterrole=cluster-admin \
  --serviceaccount=kubernetes-dashboard:dashboard-admin

# Get access token
kubectl -n kubernetes-dashboard create token dashboard-admin

# Access dashboard
kubectl proxy
# Visit: http://localhost:8001/api/v1/namespaces/kubernetes-dashboard/services/https:kubernetes-dashboard:/proxy/
```

---

## 🛑 Stopping & Starting

### Pause Deployment

```bash
# Scale everything to 0 (keeps data)
kubectl scale deployment --all --replicas=0 -n archivist
kubectl scale statefulset --all --replicas=0 -n archivist
```

### Resume Deployment

```bash
# Scale back up
kubectl scale deployment/archivist-worker --replicas=1 -n archivist
kubectl scale deployment/qdrant --replicas=1 -n archivist
kubectl scale deployment/redis --replicas=1 -n archivist
kubectl scale deployment/search-service --replicas=1 -n archivist
kubectl scale deployment/graph-service --replicas=1 -n archivist
kubectl scale statefulset/neo4j --replicas=1 -n archivist
kubectl scale statefulset/kafka --replicas=1 -n archivist
```

### Complete Cleanup

```bash
# Delete all Archivist resources
kubectl delete namespace archivist

# Data is preserved in /data/archivist/ (or $HOME/archivist-data/)

# Stop Kubernetes cluster
# For Minikube:
minikube stop

# For Docker Desktop:
# Docker Desktop → Settings → Kubernetes → Disable Kubernetes
```

---

## 🔍 Troubleshooting

### Pods Stuck in Pending

**Issue**: PVCs not binding to PVs

**Solution**:
```bash
# Check PVC status
kubectl get pvc -n archivist

# Check PV status
kubectl get pv

# Verify node name in PVs matches your cluster
kubectl get nodes

# Update k8s/base/04-local-pvs.yaml with correct node name
```

### Metrics Server Not Found (HPA Issues)

**Issue**: HPA shows "unknown" for metrics

**Solution**:
```bash
# For Minikube
minikube addons enable metrics-server

# For Docker Desktop/Kind
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# Verify
kubectl top nodes
```

### Out of Resources

**Issue**: Pods keep restarting or won't start

**Solution**:
```bash
# Check resource usage
kubectl top nodes
kubectl top pods -n archivist

# Increase Minikube resources
minikube stop
minikube start --cpus=6 --memory=12288

# Or reduce pod resource requests in deployments
```

### Cannot Access Services

**Issue**: Port forwarding doesn't work

**Solution**:
```bash
# Check if pod is running
kubectl get pods -n archivist

# Check if service exists
kubectl get svc -n archivist

# Describe service
kubectl describe svc neo4j-service -n archivist

# Try direct pod port-forward
kubectl port-forward pod/<pod-name> 7474:7474 -n archivist
```

---

## 💡 Tips for Offline Use

### Pre-pull Docker Images

```bash
# Pull images while online
docker pull neo4j:5.15-community
docker pull qdrant/qdrant:v1.7.4
docker pull redis:7.2-alpine
docker pull apache/kafka:latest

# Build Archivist images
docker build -t archivist:latest -f Dockerfile.k8s .
docker build -t archivist-search:latest -f services/search-engine/Dockerfile.k8s services/search-engine/
```

### Save Papers to Local Storage

```bash
# Mount host directory in Minikube
minikube mount $HOME/papers:/data/archivist/shared &

# Or copy directly
kubectl cp ./my_paper.pdf archivist-worker-xxx:/data/lib/my_paper.pdf -n archivist
```

### Backup Data

```bash
# Backup all data directories
tar -czf archivist-backup-$(date +%Y%m%d).tar.gz /data/archivist/

# Or backup individual services
tar -czf neo4j-backup.tar.gz /data/archivist/neo4j/
```

---

## 🎓 Learning Resources

### Understanding What's Happening

1. **Pods**: Individual containers running services
2. **Deployments**: Manage replica counts and updates
3. **StatefulSets**: For stateful services (Neo4j, Kafka)
4. **Services**: Internal DNS names for service discovery
5. **PVCs/PVs**: Persistent storage that survives pod restarts
6. **HPA**: Automatically scales pods based on CPU/memory

### Useful Commands

```bash
# See everything
kubectl get all -n archivist

# Describe a resource
kubectl describe pod/archivist-worker-xxx -n archivist

# Execute into a pod
kubectl exec -it archivist-worker-xxx -n archivist -- /bin/bash

# View events
kubectl get events -n archivist --sort-by='.lastTimestamp'

# Edit a deployment
kubectl edit deployment/archivist-worker -n archivist
```

---

## 🚀 Next Steps

1. **Process your first batch** of papers and watch autoscaling in action
2. **Monitor resources** to understand your usage patterns
3. **Experiment with scaling** - try manual and automatic
4. **Backup your data** regularly
5. **Explore the dashboard** to visualize your cluster

---

## ❓ FAQ

**Q: Why use Kubernetes on my laptop instead of Docker Compose?**
A: Kubernetes gives you autoscaling, better resource management, and it's a valuable skill to learn. Plus, it's actually quite easy with local tools like Minikube.

**Q: Will this work completely offline?**
A: Yes! After initial setup and image pulls, everything runs locally. The Gemini API still needs internet, but if you switch to a local LLM (like Ollama), it's 100% offline.

**Q: How much disk space is needed?**
A: ~5GB for Docker images, plus your paper storage. For 100 papers with reports, expect ~2-3GB total.

**Q: Can I run this on Windows?**
A: Yes! Use Docker Desktop with WSL2 or Minikube with WSL2 driver.

**Q: How do I update Archivist?**
A: Rebuild the Docker image and restart pods:
```bash
docker build -t archivist:latest -f Dockerfile.k8s .
kubectl rollout restart deployment/archivist-worker -n archivist
```

---

**Happy Local Kubernetes Learning! 🎉**
