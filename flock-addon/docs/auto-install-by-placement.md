# Auto-Install FLock Addon by Placement

This guide enables automatic addon installation through OCM Placement.

Use this only when you want the Hub to install or remove the addon automatically based on cluster labels or clustersets.

## How It Works

- `ClusterManagementAddOn` uses `installStrategy: Placements`
- `Placement` selects matching managed clusters
- OCM creates or removes `ManagedClusterAddOn` automatically

This is an alternative to manual `make enable-addon CLUSTER=...`.

## Deploy to GPU-Labeled Clusters

```bash
# [Hub]
make deploy-auto-gpu
kubectl label managedcluster cluster1 gpu=true
```

Check:

```bash
# [Hub]
kubectl -n open-cluster-management get placement flock-addon-gpu-placement -o yaml
kubectl -n cluster1 get managedclusteraddon flock-addon
```

Should see:

- the placement exists
- a `gpu=true` cluster eventually gets `managedclusteraddon/flock-addon`

## Deploy to All Clusters in the Target ClusterSet

```bash
# [Hub]
make deploy-auto-all
```

Check:

```bash
# [Hub]
kubectl -n open-cluster-management get placement flock-addon-all-placement -o yaml
kubectl get managedclusteraddons -A | rg flock-addon
```

Should see:

- the placement exists
- selected clusters eventually get `managedclusteraddon/flock-addon`

## Switch Back to Manual Mode

```bash
# [Hub]
make deploy
```

Check:

```bash
# [Hub]
kubectl get clustermanagementaddon flock-addon -o yaml | rg -n "installStrategy|type:"
```

Should see:

- `installStrategy.type` returns to `Manual`

## Per-Cluster Override with Placement

If a cluster needs different runtime defaults, create a dedicated `AddOnDeploymentConfig` and bind it explicitly through manual mode, or use a placement-specific config.

Example:

```yaml
apiVersion: addon.open-cluster-management.io/v1alpha1
kind: AddOnDeploymentConfig
metadata:
  name: flock-addon-gpu-config
  namespace: open-cluster-management
spec:
  agentInstallNamespace: flock-system
  customizedVariables:
    - name: USE_GPU
      value: "true"
```
