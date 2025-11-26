# Kubernetes Manifests

This directory contains all Kubernetes manifests for deploying Archivist with autoscaling.

## Directory Structure

```
k8s/
├── base/                           # Base manifests (environment-agnostic)
│   ├── 00-namespace.yaml          # Namespace definition
│   ├── 01-configmap.yaml          # Configuration settings
│   ├── 02-secrets.yaml            # Secrets template
│   ├── 03-pvcs.yaml               # Persistent Volume Claims
│   ├── 10-neo4j-statefulset.yaml # Neo4j StatefulSet + Service
│   ├── 11-kafka-statefulset.yaml # Kafka StatefulSet + Service
│   ├── 20-qdrant-deployment.yaml # Qdrant Deployment + Service
│   ├── 21-redis-deployment.yaml  # Redis Deployment + Service
│   ├── 30-archivist-worker-deployment.yaml  # Main worker Deployment
│   ├── 31-search-service-deployment.yaml    # Search service
│   ├── 32-graph-service-deployment.yaml     # Graph service
│   ├── 40-hpa.yaml                # HorizontalPodAutoscalers
│   └── 50-ingress.yaml            # Ingress (optional)
├── overlays/                       # Environment-specific overrides
│   ├── production/                # Production config
│   └── development/               # Development config
└── README.md                      # This file
```

## Quick Start

### Deploy Everything

```bash
# From project root
./scripts/k8s-deploy.sh build
```

### Deploy Manually

```bash
# Apply in order
kubectl apply -f k8s/base/00-namespace.yaml
kubectl apply -f k8s/base/01-configmap.yaml
kubectl apply -f k8s/base/02-secrets.yaml  # Update first!
kubectl apply -f k8s/base/03-pvcs.yaml
kubectl apply -f k8s/base/10-neo4j-statefulset.yaml
kubectl apply -f k8s/base/11-kafka-statefulset.yaml
kubectl apply -f k8s/base/20-qdrant-deployment.yaml
kubectl apply -f k8s/base/21-redis-deployment.yaml
kubectl apply -f k8s/base/30-archivist-worker-deployment.yaml
kubectl apply -f k8s/base/31-search-service-deployment.yaml
kubectl apply -f k8s/base/32-graph-service-deployment.yaml
kubectl apply -f k8s/base/40-hpa.yaml
```

## Manifest Details

### 00-namespace.yaml
Creates the `archivist` namespace for isolating all resources.

### 01-configmap.yaml
Contains application configuration:
- Service URLs (Neo4j, Qdrant, Redis, Kafka)
- Processing settings
- Graph database config
- Vector database config

**Note**: Update service URLs if changing service names.

### 02-secrets.yaml
**⚠️ IMPORTANT**: This is a template. Create actual secrets:

```bash
kubectl create secret generic archivist-secrets \
  --from-literal=GEMINI_API_KEY='your-api-key' \
  --from-literal=NEO4J_PASSWORD='secure-password' \
  -n archivist
```

### 03-pvcs.yaml
Creates persistent storage:
- **neo4j-data-pvc**: 10Gi for Neo4j graph data
- **qdrant-storage-pvc**: 5Gi for vector data
- **redis-data-pvc**: 2Gi for cache
- **kafka-data-pvc**: 10Gi for message queue
- **archivist-data-pvc**: 20Gi shared storage (ReadWriteMany)

**Note**: Update `storageClassName` for your cloud provider:
- GKE: `pd-ssd` or `pd-standard`
- EKS: `gp3` or `gp2`
- AKS: `managed-premium`

### 10-neo4j-statefulset.yaml
Neo4j graph database (StatefulSet):
- **Replicas**: 1 (Community Edition)
- **Resources**: 1-4Gi memory, 500m-2 CPU
- **Ports**: 7474 (HTTP), 7687 (Bolt)
- **Plugins**: APOC, Graph Data Science

### 11-kafka-statefulset.yaml
Apache Kafka message broker (StatefulSet):
- **Replicas**: 1
- **Resources**: 1-2Gi memory, 500m-1 CPU
- **Ports**: 9092 (internal), 9094 (external)

### 20-qdrant-deployment.yaml
Qdrant vector database (Deployment):
- **Replicas**: 2 (can scale with HPA)
- **Resources**: 512Mi-2Gi memory, 250m-1 CPU
- **Ports**: 6333 (HTTP), 6334 (gRPC)
- **Autoscaling**: 1-5 replicas

### 21-redis-deployment.yaml
Redis cache (Deployment):
- **Replicas**: 1
- **Resources**: 256Mi-1Gi memory, 100m-500m CPU
- **Port**: 6379
- **Config**: AOF persistence, LRU eviction

### 30-archivist-worker-deployment.yaml
Main processing workers (Deployment):
- **Replicas**: 3 (initial), 2-20 (HPA range)
- **Resources**: 1-4Gi memory, 500m-2 CPU
- **Ports**: 8080 (API), 9090 (metrics)
- **Autoscaling**: CPU/memory based

### 31-search-service-deployment.yaml
Academic paper search service (Deployment):
- **Replicas**: 2 (initial), 2-10 (HPA range)
- **Resources**: 256Mi-1Gi memory, 250m-1 CPU
- **Port**: 8000
- **API**: FastAPI with arXiv, OpenReview, ACL

### 32-graph-service-deployment.yaml
Graph operations service (Deployment):
- **Replicas**: 2 (initial), 2-10 (HPA range)
- **Resources**: 512Mi-2Gi memory, 250m-1 CPU
- **Port**: 8081
- **Features**: Citation extraction, graph analytics

### 40-hpa.yaml
Horizontal Pod Autoscalers for:
1. **archivist-worker-hpa**
   - Min: 2, Max: 20
   - Triggers: CPU > 70%, Memory > 80%
   - Fast scale-up, gradual scale-down

2. **qdrant-hpa**
   - Min: 1, Max: 5
   - Triggers: CPU > 75%, Memory > 80%

3. **search-service-hpa**
   - Min: 2, Max: 10
   - Triggers: CPU > 70%, Memory > 75%

4. **graph-service-hpa**
   - Min: 2, Max: 10
   - Triggers: CPU > 70%, Memory > 80%

### 50-ingress.yaml
Optional Ingress for external access:
- **Ingress Controller**: nginx (default)
- **TLS**: cert-manager integration
- **Routes**:
  - `/neo4j` → Neo4j Browser
  - `/qdrant` → Qdrant Dashboard
  - `/api/search` → Search API
  - `/api/graph` → Graph API
  - `/api/worker` → Worker API

**Note**: Update hostname and TLS settings before deploying.

## Resource Requirements

### Minimum Cluster

- **Nodes**: 3
- **Total CPU**: 8 cores
- **Total Memory**: 16GB RAM
- **Storage**: 50GB+ persistent

### Recommended Cluster

- **Nodes**: 5+
- **Total CPU**: 16 cores
- **Total Memory**: 32GB RAM
- **Storage**: 100GB+ persistent

## Scaling

### Manual Scaling

```bash
# Scale workers to 10 replicas
kubectl scale deployment/archivist-worker --replicas=10 -n archivist

# Scale search service
kubectl scale deployment/search-service --replicas=5 -n archivist
```

### Autoscaling

Autoscaling is automatic based on:
- CPU utilization
- Memory utilization
- Custom metrics (optional)

Monitor autoscaling:
```bash
# Watch HPA status
watch kubectl get hpa -n archivist

# Detailed HPA info
kubectl describe hpa archivist-worker-hpa -n archivist
```

## Updating

### Update Configuration

```bash
# Edit ConfigMap
kubectl edit configmap archivist-config -n archivist

# Restart pods to pick up changes
kubectl rollout restart deployment/archivist-worker -n archivist
```

### Update Secrets

```bash
# Update secret
kubectl create secret generic archivist-secrets \
  --from-literal=GEMINI_API_KEY='new-key' \
  --from-literal=NEO4J_PASSWORD='new-password' \
  -n archivist \
  --dry-run=client -o yaml | kubectl apply -f -

# Restart pods
kubectl rollout restart deployment/archivist-worker -n archivist
```

### Update Images

```bash
# Update image version
kubectl set image deployment/archivist-worker \
  worker=your-registry/archivist:v2.0 \
  -n archivist

# Watch rollout
kubectl rollout status deployment/archivist-worker -n archivist
```

## Monitoring

### Resource Usage

```bash
# Pod resource usage
kubectl top pods -n archivist

# Node resource usage
kubectl top nodes
```

### Logs

```bash
# Worker logs
kubectl logs -f deployment/archivist-worker -n archivist

# Search service logs
kubectl logs -f deployment/search-service -n archivist

# All logs for a pod
kubectl logs pod-name -n archivist --all-containers
```

### Events

```bash
# Recent events
kubectl get events -n archivist --sort-by='.lastTimestamp'

# Watch events
kubectl get events -n archivist --watch
```

## Troubleshooting

### Pods Not Starting

```bash
# Check pod status
kubectl get pods -n archivist

# Describe pod
kubectl describe pod <pod-name> -n archivist

# Check logs
kubectl logs <pod-name> -n archivist
```

### PVCs Not Binding

```bash
# Check PVC status
kubectl get pvc -n archivist

# Check storage classes
kubectl get sc

# Describe PVC
kubectl describe pvc <pvc-name> -n archivist
```

### Services Not Accessible

```bash
# Check services
kubectl get svc -n archivist

# Check endpoints
kubectl get endpoints -n archivist

# Test from pod
kubectl exec -it <pod-name> -n archivist -- curl http://service-name:port/health
```

## Cleanup

### Delete Everything

```bash
# Using management script
./scripts/k8s-manage.sh delete

# Or manually
kubectl delete namespace archivist
```

### Keep Data, Remove Pods

```bash
# Delete deployments and statefulsets
kubectl delete deployments --all -n archivist
kubectl delete statefulsets --all -n archivist

# PVCs will remain for data persistence
```

## Support

- Full Guide: [docs/KUBERNETES_DEPLOYMENT.md](../docs/KUBERNETES_DEPLOYMENT.md)
- Deployment Script: [scripts/k8s-deploy.sh](../scripts/k8s-deploy.sh)
- Management Script: [scripts/k8s-manage.sh](../scripts/k8s-manage.sh)

For issues: https://github.com/yourusername/Archivist/issues
