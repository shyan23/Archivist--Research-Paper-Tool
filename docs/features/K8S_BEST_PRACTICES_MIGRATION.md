# Kubernetes Best Practices Migration Guide

## Summary of Changes

We've implemented **Kubernetes best practices** for your Archivist deployment:

### ✅ What Changed

1. **Worker Node Isolation**
   - Storage (PVs) now pinned to **worker nodes** (not control-plane)
   - Workloads automatically schedule on workers
   - Control-plane remains tainted and protected

2. **Automatic Cleanup**
   - Setup script now destroys existing deployments before creating new ones
   - Ensures clean state on every run
   - Prevents resource conflicts

3. **Kind-Specific Optimizations**
   - Proper `extraMounts` configuration for worker nodes
   - Correct path mapping: `$HOME/archivist-data` → `/data` (in container)

4. **Fixed Dockerfile**
   - Updated binary path from `./cmd/rph` to `./cmd/main`
   - Binary renamed to `archivist` for consistency

---

## Architecture: Before vs After

### ❌ Before (Quick Fix - Not Production Ready)

```
┌─────────────────────────────────────┐
│   tkb-control-plane (UNTAINTED)     │
│                                     │
│   ❌ Neo4j pods running here        │
│   ❌ Kafka pods running here        │
│   ❌ Application pods running here  │
│   ❌ Storage mounted here           │
└─────────────────────────────────────┘

┌─────────────────────────────────────┐
│   tkb-worker (IDLE)                 │
│                                     │
│   ⚠️  No storage mounted            │
│   ⚠️  No pods running               │
└─────────────────────────────────────┘
```

### ✅ After (Best Practice - Production Ready)

```
┌─────────────────────────────────────┐
│   tkb-control-plane (TAINTED)       │
│                                     │
│   ✅ Only system pods (API, etc)    │
│   ✅ No storage                     │
│   ✅ No application workloads       │
└─────────────────────────────────────┘

┌─────────────────────────────────────┐
│   tkb-worker (ACTIVE)               │
│                                     │
│   ✅ Neo4j pods here                │
│   ✅ Kafka pods here                │
│   ✅ Application pods here          │
│   ✅ Storage mounted: /data         │
└─────────────────────────────────────┘
```

---

## Migration Path

You have **two options** for applying these changes:

### Option A: Fresh Cluster (Recommended - 5 minutes)

**Use this if:** You're okay recreating the cluster from scratch.

```bash
# Step 1: Recreate cluster with proper configuration
./scripts/recreate-kind-cluster-proper.sh

# Step 2: Deploy Archivist with new setup
./scripts/k8s-local-setup.sh

# Step 3: Build and load Docker images
./scripts/build-and-load-k8s-images.sh
```

**What happens:**
- Deletes existing `tkb` cluster
- Creates new cluster with `kind-config.yaml` (worker mounts configured)
- Control-plane stays tainted (best practice)
- Storage auto-configures on worker nodes

---

### Option B: Keep Current Cluster (Manual - 10 minutes)

**Use this if:** You want to keep the existing cluster but fix the configuration.

**⚠️ Current State:**
- You already removed the control-plane taint (quick fix)
- Pods are running on control-plane (works but not ideal)
- Worker nodes don't have storage mounted

**Steps to migrate:**

```bash
# 1. Delete all Archivist resources
kubectl delete namespace archivist

# 2. Delete persistent volumes
kubectl delete pv neo4j-pv qdrant-pv redis-pv kafka-pv archivist-data-pv

# 3. Stop the cluster
kind delete cluster --name tkb

# 4. Now follow Option A
./scripts/recreate-kind-cluster-proper.sh
```

---

## Files Modified

### New Files Created:
- `kind-config.yaml` - Cluster configuration with worker mounts
- `scripts/recreate-kind-cluster-proper.sh` - Cluster recreation script
- `scripts/build-and-load-k8s-images.sh` - Docker image builder
- `K8S_BEST_PRACTICES_MIGRATION.md` - This guide

### Modified Files:
- `scripts/k8s-local-setup.sh`
  - Auto-cleanup existing deployments
  - Worker node selection (not control-plane)
  - Kind-specific path handling (`/data`)

- `Dockerfile`
  - Fixed build path: `./cmd/main` (was `./cmd/rph`)
  - Binary renamed to `archivist`

---

## Verification After Migration

After running the migration, verify everything is correct:

```bash
# 1. Check nodes and taints
kubectl get nodes
kubectl get nodes -o custom-columns=NAME:.metadata.name,TAINTS:.spec.taints

# Expected output:
# tkb-control-plane should show taint: node-role.kubernetes.io/control-plane:NoSchedule
# tkb-worker should show: <none>
# tkb-worker2 should show: <none>

# 2. Check pod placement
kubectl get pods -n archivist -o wide

# Expected: All pods should be on tkb-worker or tkb-worker2
# None should be on tkb-control-plane

# 3. Check storage
kubectl get pv
kubectl get pvc -n archivist

# Expected: All PVCs should be Bound
# Node affinity should point to worker node
```

---

## Why This Matters

### Production Alignment
- This setup mirrors real production Kubernetes clusters
- Control-planes are always tainted in production
- Separation of concerns: brain vs. workers

### Crash Resilience
- If a worker crashes, control-plane keeps cluster alive
- Can drain workers for maintenance without cluster downtime
- Better resource isolation

### Learning
- Prepares you for real cloud deployments (GKE, EKS, AKS)
- Teaches proper node affinity and taints/tolerations
- Understanding of Kubernetes scheduling

---

## Quick Reference

### Current Cluster Status
```bash
# Before migration (current state):
✅ Cluster: kind-tkb (3 nodes)
❌ Control-plane: UNTAINTED (pods running here)
❌ Workers: No storage mounted
❌ PVs: Pinned to control-plane

# After migration (target state):
✅ Cluster: kind-tkb (3 nodes)
✅ Control-plane: TAINTED (system pods only)
✅ Workers: Storage mounted at /data
✅ PVs: Pinned to worker nodes
✅ All app pods: Running on workers
```

### Key Commands
```bash
# Check current taint
kubectl describe node tkb-control-plane | grep -A 3 Taints

# Manually add taint (if needed)
kubectl taint nodes tkb-control-plane node-role.kubernetes.io/control-plane:NoSchedule

# Manually remove taint (quick fix - not recommended)
kubectl taint nodes tkb-control-plane node-role.kubernetes.io/control-plane:NoSchedule-

# Check where pods are running
kubectl get pods -n archivist -o wide
```

---

## Next Steps

1. **Choose your migration path** (Option A or Option B above)
2. **Run the migration scripts**
3. **Verify the setup** using the verification commands
4. **Test with a sample paper**:
   ```bash
   cp sample_paper.pdf ~/archivist-data/shared/lib/
   kubectl logs -f deployment/archivist-worker -n archivist
   ```

---

## Troubleshooting

### Issue: "No worker nodes found"
**Solution:** Your cluster doesn't have worker nodes configured. Use Option A to recreate with proper config.

### Issue: Pods still pending after migration
**Solution:**
```bash
kubectl describe pod <pod-name> -n archivist
# Look for "Events" section - it will tell you why
```

### Issue: PVCs not binding
**Solution:**
```bash
kubectl get pv  # Check if node affinity matches worker node
kubectl describe pvc <pvc-name> -n archivist
```

---

## Support

If you encounter issues:
1. Check the "Verification" section above
2. Look at pod events: `kubectl describe pod <pod-name> -n archivist`
3. Check logs: `kubectl logs <pod-name> -n archivist`

---

**Ready to migrate? Start with Option A (recommended):**

```bash
./scripts/recreate-kind-cluster-proper.sh
```
