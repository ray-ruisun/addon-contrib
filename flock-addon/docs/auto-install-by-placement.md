# Auto-Install FLock Addon by Placement

This guide enables automatic addon installation through OCM Placement.

Use this only when you want the Hub to install or remove the addon automatically based on cluster labels or clustersets.
Auto-install targets accept the same image overrides as manual deploy (`IMAGE_OWNER`, `IMAGE_TAG`,
`FLOCK_ALLIANCE_IMAGE`, `IMAGE_PULL_SECRET`) and optional blockchain overrides (`TASK_ADDRESS`, `RPC`, `TOKEN_ADDRESS`).

## How It Works

- `ClusterManagementAddOn` uses `installStrategy: Placements`
- `Placement` selects matching managed clusters
- OCM creates or removes `ManagedClusterAddOn` automatically

This is an alternative to manual `make enable-addon CLUSTER=...`.

## Deploy to GPU-Labeled Clusters

```bash
# [Hub]
IMAGE_OWNER='ray-ruisun' IMAGE_TAG='<git-sha-or-release-tag>' \
TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924' make deploy-auto-gpu
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
IMAGE_OWNER='ray-ruisun' IMAGE_TAG='<git-sha-or-release-tag>' \
TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924' make deploy-auto-all
```

`deploy-auto-all` is dynamic by default. It installs the GPU template on
`gpu=true` clusters and the CPU template on the rest.

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

If a cluster needs different runtime defaults, create a dedicated `AddOnDeploymentConfig` and bind it explicitly through manual mode.
Placement mode uses the built-in CPU/GPU config pair automatically:

- `flock-addon-config` for the CPU template
- `flock-addon-gpu-config` for the GPU template

`customizedVariables` are still the runtime injection mechanism. What was removed
were the old chart value paths that used to feed some of those variables.
`TASK_ADDRESS`, `USE_GPU`, `STORAGE_BACKEND`, and `NO_INCENTIVE` stay authoritative
from OCM. In testnet mode, `BLOCKCHAIN_RPC` comes from the mounted `.env`.
When `STORAGE_BACKEND=local`, `BLOCKCHAIN_RPC`, `TOKEN_ADDRESS`, and
`LOCAL_STORAGE_DIR` also stay authoritative from OCM when non-empty.
`NUM_PARTICIPANTS` is forced from OCM only when `STORAGE_BACKEND=local`; other
runtime keys can come from the mounted `.env` on each cluster node.

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
