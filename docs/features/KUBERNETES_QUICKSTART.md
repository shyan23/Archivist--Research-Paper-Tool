# Kubernetes Deployment - One Command Setup

## 🚀 Quick Start (One Command!)

```bash
./scripts/k8s-local-setup.sh
```

That's it! This single command now:

1. ✅ **Validates** your cluster configuration
2. ✅ **Offers to recreate** cluster with best practices (if needed)
3. ✅ **Destroys all existing pods** automatically
4. ✅ **Creates fresh deployment** from scratch
5. ✅ **Configures storage** on worker nodes (best practice)
6. ✅ **Keeps control-plane tainted** (production-ready)

---

## What Happens When You Run It

### Step 1: Cluster Validation

The script checks if your Kind cluster follows best practices:

```
✅ Worker nodes have /data mount?
✅ Control-plane is tainted?
✅ Storage accessible on workers?
```

**If issues found:**
- Script will **ask** if you want to recreate the cluster
- Answer **`y`** for best practices (recommended)
- Answer **`n`** to use quick-fix mode (works but not ideal)

### Step 2: Automatic Cleanup

If you have existing pods, the script automatically:

```
🧹 Cleaning up ALL existing resources...
  [1/5] Deleting HPAs
  [2/5] Deleting deployments and statefulsets
  [3/5] Deleting services
  [4/5] Deleting PVCs and PVs
  [5/5] Waiting for pods to terminate
✅ Complete cleanup finished
```

### Step 3: Fresh Deployment

Creates everything from scratch:

```
Neo4j → Kafka → Qdrant → Redis → Workers → Services
```

---

## First Time Setup

If this is your **first time** running the script on a **new cluster**:

```bash
# Make sure you're in the Archivist root directory
cd ~/Desktop/Code/Archivist

# Run the setup
./scripts/k8s-local-setup.sh
```

**You'll be asked:**
1. To recreate cluster with best practices → **Answer: y**
2. Enter your Gemini API key → **Paste your key**
3. Enter Neo4j password → **Default: password** (or set your own)

---

## Subsequent Runs

Every time you run `./scripts/k8s-local-setup.sh` again:

- **Old pods**: Automatically destroyed
- **Old PVs**: Deleted and recreated
- **Fresh state**: Brand new deployment

**Perfect for:**
- Testing configuration changes
- Applying updates
- Starting clean after experiments

---

## After Deployment

### Build & Load Docker Images

The base services (Neo4j, Kafka, Redis, Qdrant) will start automatically.

For the **custom services** (archivist-worker, graph-service, search-service):

```bash
./scripts/build-and-load-k8s-images.sh
```

This builds:
1. `archivist:latest` (worker)
2. `archivist-graph:latest` (graph service)
3. `archivist-search:latest` (search service)

And loads them into Kind.

### Check Status

```bash
# Watch pods come up
watch kubectl get pods -n archivist

# Check services
kubectl get svc -n archivist

# Check autoscaling
kubectl get hpa -n archivist
```

### Access Services

```bash
# Neo4j
kubectl port-forward svc/neo4j-service 7474:7474 -n archivist
# Open: http://localhost:7474

# Qdrant
kubectl port-forward svc/qdrant-service 6333:6333 -n archivist
# Open: http://localhost:6333/dashboard

# Search API
kubectl port-forward svc/search-service 8000:8000 -n archivist
# Open: http://localhost:8000/docs
```

---

## Architecture (After Setup)

### Best Practice Mode (Recommended)

```
┌─────────────────────────────────┐
│  tkb-control-plane (TAINTED)    │
│  ✅ System pods only             │
│  ✅ No storage                   │
└─────────────────────────────────┘

┌─────────────────────────────────┐
│  tkb-worker (ACTIVE)            │
│  ✅ All app pods                 │
│  ✅ Storage at /data             │
│  ✅ Auto-scales 1-4 replicas     │
└─────────────────────────────────┘
```

### Quick-Fix Mode (Fallback)

```
┌─────────────────────────────────┐
│  tkb-control-plane (UNTAINTED)  │
│  ⚠️  App pods running here       │
│  ⚠️  Storage here                │
│  ℹ️  Works but not production    │
└─────────────────────────────────┘
```

---

## Common Workflows

### Scenario 1: First Time User

```bash
# Clone and enter directory
git clone <repo>
cd Archivist

# One command setup
./scripts/k8s-local-setup.sh
# Answer 'y' to recreate cluster
# Enter API key and password

# Build images
./scripts/build-and-load-k8s-images.sh

# Done! Check status
kubectl get pods -n archivist
```

### Scenario 2: Making Changes & Redeploying

```bash
# Edit Kubernetes manifests
vim k8s/base/30-archivist-worker-deployment.yaml

# Redeploy (auto-cleanup + fresh deployment)
./scripts/k8s-local-setup.sh
# Answer 'n' to keep existing cluster
```

### Scenario 3: Code Changes to Services

```bash
# Make code changes
vim internal/worker/pool.go

# Rebuild images
./scripts/build-and-load-k8s-images.sh

# Restart pods to pick up new images
kubectl delete pod -n archivist -l app=archivist-worker
```

### Scenario 4: Nuclear Option (Start Fresh)

```bash
# Delete everything
kind delete cluster --name tkb

# Recreate from scratch
./scripts/k8s-local-setup.sh
# Answer 'y' to create cluster

# Build images
./scripts/build-and-load-k8s-images.sh
```

---

## Troubleshooting

### Pods Stuck in Pending

```bash
# Check events
kubectl describe pod <pod-name> -n archivist

# Common issues:
# - PVC not bound → Check PV node affinity
# - Node not ready → Check if worker has /data mount
# - Image not found → Run build-and-load-k8s-images.sh
```

### "Worker node missing /data mount"

**Solution:**
```bash
# Let script recreate cluster
./scripts/k8s-local-setup.sh
# Answer 'y' when asked to recreate
```

### Images Pull Failed

```bash
# Build and load images into Kind
./scripts/build-and-load-k8s-images.sh

# Verify
docker images | grep archivist
```

### Check What's Wrong

```bash
# Pod logs
kubectl logs <pod-name> -n archivist

# Events in namespace
kubectl get events -n archivist --sort-by='.lastTimestamp'

# Describe everything
kubectl describe all -n archivist
```

---

## Key Files

- `kind-config.yaml` - Cluster configuration (worker mounts)
- `scripts/k8s-local-setup.sh` - **Main deployment script** ⭐
- `scripts/build-and-load-k8s-images.sh` - Image builder
- `k8s/base/` - All Kubernetes manifests
  - `00-namespace.yaml` - Namespace
  - `04-local-pvs.yaml` - Storage volumes
  - `10-neo4j-statefulset.yaml` - Neo4j
  - `11-kafka-statefulset.yaml` - Kafka
  - `20-qdrant-deployment.yaml` - Qdrant
  - `21-redis-deployment.yaml` - Redis
  - `30-archivist-worker-deployment.yaml` - Workers
  - `31-search-service-deployment.yaml` - Search API
  - `32-graph-service-deployment.yaml` - Graph processor
  - `40-hpa.yaml` - Autoscaling rules

---

## Summary

**One command does it all:**

```bash
./scripts/k8s-local-setup.sh
```

**Features:**
- ✅ Auto-detects cluster issues
- ✅ Offers to fix them
- ✅ Destroys old pods automatically
- ✅ Creates fresh deployment
- ✅ Best practices by default
- ✅ Quick-fix mode as fallback

**Production-ready from day one!** 🚀
