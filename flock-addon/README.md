# FLock Addon

FLock addon deploys **FLockAlliance** to managed clusters as a direct client workload through Open Cluster Management (OCM).

- Runtime mode is fixed to `local`
- `FLocKit` is not deployed as a separate addon workload
- Each managed cluster runs one `flock-agent` Deployment
- Runtime configuration is loaded from a mounted node directory, usually `/data/flock-client`

## Documentation

- [Install FLock Addon](docs/install-flock-addon.md)
- [Auto-Install by Placement](docs/auto-install-by-placement.md)
- [Troubleshooting](docs/troubleshooting.md)

## What Runs Where

### Hub cluster

- Stores `ClusterManagementAddOn`, `AddOnTemplate`, and `AddOnDeploymentConfig`
- Deploys or updates shared addon settings
- Enables the addon for selected managed clusters

### Managed cluster

- Receives `ManifestWork`
- Runs `flock-agent` in namespace `flock-system`

### Managed cluster node

- Provides the mounted host path, usually `/data/flock-client`
- Stores `.env` and any local datasets/files used by `FLockAlliance`

## Runtime Model

Each managed cluster gets one Pod with one container:

- Deployment: `flock-agent`
- Container: `flock-alliance-client`
- Container mount path: `/data`
- Default node path: `/data/flock-client`
- Default env file inside container: `/data/.env`
- Effective env file on node: `/data/flock-client/.env`

Path rules:

- Do not use `~` in `hostPath`
- Use an absolute path such as `/data/flock-client`
- The same host path must exist on every node that may schedule the Pod

## Image Selection

Chart fallback image:

- `ghcr.io/flock-io/fl-alliance-client:latest`

The `make` flow supports environment-variable based image overrides:

- `IMAGE_REGISTRY`, default `ghcr.io`
- `IMAGE_OWNER`, default `flock-io`
- `IMAGE_NAME`, default `fl-alliance-client`
- `IMAGE_TAG`, default `latest`
- `IMAGE_PULL_SECRET`, optional managed-cluster image pull secret name
- `FLOCK_ALLIANCE_IMAGE`, overrides all of the above

Example:

```bash
# [Hub]
IMAGE_OWNER='ray-ruisun' IMAGE_TAG='latest' make deploy
```

Or:

```bash
# [Hub]
FLOCK_ALLIANCE_IMAGE='ghcr.io/ray-ruisun/fl-alliance-client:latest' make deploy
```

Recommended explicit export form:

```bash
# [Hub]
export IMAGE_REGISTRY='ghcr.io'
export IMAGE_OWNER='ray-ruisun'
export IMAGE_NAME='fl-alliance-client'
export IMAGE_TAG='latest'
export FLOCK_ALLIANCE_IMAGE="${IMAGE_REGISTRY}/${IMAGE_OWNER}/${IMAGE_NAME}:${IMAGE_TAG}"
```

Use it for deployment:

```bash
# [Hub]
make deploy-testnet TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
```

Check:

```bash
# [Hub]
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config -o yaml | rg -n "FLOCK_ALLIANCE_IMAGE|value"
```

Should see:

- `FLOCK_ALLIANCE_IMAGE` matches `${IMAGE_REGISTRY}/${IMAGE_OWNER}/${IMAGE_NAME}:${IMAGE_TAG}`

If the image is private, also set:

```bash
# [Hub]
export IMAGE_PULL_SECRET='ghcr-pull'
```

If the selected registry is private, also configure `image.pullSecrets`.

## Quick Start: Testnet Mode

This is the default deployment mode.

Behavior:

- `TASK_ADDRESS` must be passed from the Hub at deploy time
- `BLOCKCHAIN_RPC` and `TOKEN_ADDRESS` are read from each cluster node `.env`
- `PRIVATE_KEY` and `HF_TOKEN` are read from each cluster node `.env`
- GPU runtime is enabled by default through `deploymentConfig.runtime.useGpu=true`

### 1) Prepare the node path

Run on every managed cluster node that may host the addon Pod.

```bash
# [Each Managed Cluster Node]
sudo mkdir -p /data/flock-client
sudo chmod 755 /data
sudo chown -R ubuntu:ubuntu /data/flock-client
sudo chmod -R u+rwX /data/flock-client
```

Check:

```bash
# [Each Managed Cluster Node]
ls -ld /data /data/flock-client
```

Should see:

- `/data` exists
- `/data/flock-client` exists
- your login user can read and write `/data/flock-client`

If your login user is not `ubuntu`, replace `ubuntu:ubuntu` with your actual user and group.

### 2) Create the node `.env`

Create this file on every managed cluster node:

```text
/data/flock-client/.env
```

Example testnet `.env`:

```dotenv
PRIVATE_KEY=0x...
HF_TOKEN=hf_...
BLOCKCHAIN_RPC=https://sepolia.base.org
TOKEN_ADDRESS=0x...
STORAGE_BACKEND=s3
LOCAL_STORAGE_DIR=/data/shared
```

Check:

```bash
# [Each Managed Cluster Node]
ls -l /data/flock-client/.env
sed -n '1,20p' /data/flock-client/.env
```

Should see:

- `.env` exists at `/data/flock-client/.env`
- required keys are present

### 3) Deploy the addon definition on the Hub

Run on the Hub.

```bash
# [Hub]
cd flock-addon
make deploy-testnet TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
```

If you want a different image owner:

```bash
# [Hub]
cd flock-addon
IMAGE_OWNER='ray-ruisun' make deploy-testnet TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
```

Check:

```bash
# [Hub]
kubectl get clustermanagementaddon flock-addon
kubectl -n open-cluster-management get addontemplate flock-addon
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config -o yaml | rg -n "TASK_ADDRESS|FLOCK_ALLIANCE_IMAGE|value"
```

Should see:

- `clustermanagementaddon/flock-addon` exists
- `addontemplate/flock-addon` exists
- `addondeploymentconfig/flock-addon-config` exists
- `TASK_ADDRESS` matches the value you passed
- `FLOCK_ALLIANCE_IMAGE` matches the image you expect

### 4) Enable the addon on a managed cluster

Run on the Hub.

```bash
# [Hub]
make enable-addon CLUSTER=cluster1
```

Check:

```bash
# [Hub]
kubectl -n cluster1 get managedclusteraddon flock-addon -o yaml
kubectl -n cluster1 get manifestwork
```

Should see:

- `managedclusteraddon/flock-addon` exists
- `spec.configs` includes `flock-addon` and `flock-addon-config`
- a `ManifestWork` for `flock-addon` appears

### 5) Verify the runtime on the managed cluster

Run against the managed cluster context.

```bash
# [Managed Cluster context]
kubectl -n flock-system get deploy,pod
kubectl -n flock-system logs deploy/flock-agent -c flock-alliance-client --tail=100
```

Should see:

- `deployment/flock-agent` exists
- Pod becomes `Running`
- logs show `FLockAlliance` startup rather than image pull or crash errors

## Local Chain Mode

Use local chain mode only if the blockchain endpoint is reachable from the addon Pod.

Important:

- do not use `127.0.0.1` unless the chain runs in the same Pod
- prefer a node IP or Kubernetes Service DNS reachable from `flock-agent`

Example `.env`:

```dotenv
PRIVATE_KEY=0x...
HF_TOKEN=hf_...
BLOCKCHAIN_RPC=http://<node-ip-or-service>:8545
TOKEN_ADDRESS=0x...
TASK_ADDRESS=0x...
STORAGE_BACKEND=local
LOCAL_STORAGE_DIR=/data/shared
```

Deploy command:

```bash
# [Hub]
helm upgrade --install flock-addon charts/flock-addon \
  --set agent.dataVolume.hostPath='/data/flock-client' \
  --set deploymentConfig.storage.backend='local' \
  --set deploymentConfig.storage.localSharedDir='/data/shared'
```

Check:

```bash
# [Hub]
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config -o yaml | rg -n "STORAGE_BACKEND|LOCAL_STORAGE_DIR|value"
```

Should see:

- `STORAGE_BACKEND` is `local`
- `LOCAL_STORAGE_DIR` is `/data/shared`

## Update Task Address

When a new onchain task is created, update only `TASK_ADDRESS`.

```bash
# [Hub]
make update-task TASK_ADDRESS='0x<NEW_TASK_ADDRESS>'
```

Check:

```bash
# [Hub]
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config -o yaml | rg -n "TASK_ADDRESS|value"
```

Should see:

- `TASK_ADDRESS` matches the new value

If the addon is already enabled, the managed cluster workload should reconcile automatically. If you want to force a refresh:

```bash
# [Hub]
make disable-addon CLUSTER=cluster1
make enable-addon CLUSTER=cluster1
```

Check:

```bash
# [Hub]
kubectl -n cluster1 get manifestwork
```

Should see:

- the `flock-addon` ManifestWork exists after re-enable

## Image Pull Troubleshooting

If the managed cluster Pod is stuck in `ImagePullBackOff` or `ErrImagePull`, inspect the Pod events first.

```bash
# [Managed Cluster context]
kubectl -n flock-system describe pod -l app.kubernetes.io/name=flock-addon
```

Check:

```bash
# [Hub]
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config -o yaml | rg -n "FLOCK_ALLIANCE_IMAGE|value"
```

Should see:

- the Pod events show the exact pull failure
- `FLOCK_ALLIANCE_IMAGE` matches the intended image

If the image is wrong, redeploy with an explicit override:

```bash
# [Hub]
IMAGE_OWNER='ray-ruisun' IMAGE_TAG='latest' make deploy-testnet TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
make disable-addon CLUSTER=cluster1
make enable-addon CLUSTER=cluster1
```

Check:

```bash
# [Managed Cluster context]
kubectl -n flock-system get deploy,pod
```

Should see:

- the Pod is recreated with the updated image

If the Pod event says `unauthorized` or `denied`, the registry needs credentials.

Create the registry secret on the managed cluster:

```bash
# [Managed Cluster context]
kubectl -n flock-system create secret docker-registry ghcr-creds \
  --docker-server=ghcr.io \
  --docker-username='<github-user>' \
  --docker-password='<github-token>' \
  --docker-email='<email>'
```

Then redeploy from the Hub with pull secrets:

```bash
# [Hub]
helm upgrade --install flock-addon charts/flock-addon \
  --set image.repository='ghcr.io/ray-ruisun/fl-alliance-client' \
  --set image.tag='latest' \
  --set image.pullSecrets[0]='ghcr-creds'
```

Check:

```bash
# [Managed Cluster context]
kubectl -n flock-system get secret ghcr-creds
kubectl -n flock-system get deploy flock-agent -o yaml | rg -n "imagePullSecrets|ghcr-creds"
```

Should see:

- `ghcr-creds` exists
- `deployment/flock-agent` references `imagePullSecrets`

## OCM Distribution Troubleshooting

If `make deploy-testnet` succeeds on the Hub but nothing runs on the managed cluster, check the OCM distribution chain from Hub to managed cluster.

### Hub-side checks

```bash
# [Hub]
kubectl get clustermanagementaddon flock-addon
kubectl -n cluster1 get managedclusteraddon flock-addon -o yaml
kubectl -n cluster1 get manifestwork
```

Should see:

- `clustermanagementaddon/flock-addon` exists
- `managedclusteraddon/flock-addon` exists in the managed cluster namespace on the Hub
- a `ManifestWork` exists for `flock-addon`

If `ManagedClusterAddOn` exists but no `ManifestWork` appears, re-enable the addon:

```bash
# [Hub]
make disable-addon CLUSTER=cluster1
make enable-addon CLUSTER=cluster1
```

Check:

```bash
# [Hub]
kubectl -n cluster1 get managedclusteraddon flock-addon -o yaml
kubectl -n cluster1 get manifestwork
```

Should see:

- `spec.configs` contains both `flock-addon` and `flock-addon-config`
- `ManifestWork` appears after re-enable

### Managed-cluster checks

```bash
# [Managed Cluster context]
kubectl get ns flock-system
kubectl -n flock-system get deploy,pod
```

Should see:

- namespace `flock-system` exists
- `flock-agent` Deployment exists

## Per-Cluster Override

If one cluster needs different defaults, create a dedicated `AddOnDeploymentConfig` and reference it from that cluster's `ManagedClusterAddOn`.

Example `AddOnDeploymentConfig`:

```yaml
apiVersion: addon.open-cluster-management.io/v1alpha1
kind: AddOnDeploymentConfig
metadata:
  name: flock-addon-config-cluster1
  namespace: open-cluster-management
spec:
  agentInstallNamespace: flock-system
  customizedVariables:
    - name: FLOCK_ALLIANCE_ENV_FILE
      value: /data/.env
    - name: BLOCKCHAIN_RPC
      value: ""
    - name: TOKEN_ADDRESS
      value: ""
    - name: TASK_ADDRESS
      value: ""
    - name: DATA_PATH
      value: /data
    - name: HOST_DATA_PATH
      value: /data/flock-client
```

Example `ManagedClusterAddOn` reference:

```yaml
apiVersion: addon.open-cluster-management.io/v1alpha1
kind: ManagedClusterAddOn
metadata:
  name: flock-addon
  namespace: cluster1
  annotations:
    addon.open-cluster-management.io/v1alpha1-install-namespace: flock-system
spec:
  configs:
    - group: addon.open-cluster-management.io
      resource: addontemplates
      name: flock-addon
    - group: addon.open-cluster-management.io
      resource: addondeploymentconfigs
      name: flock-addon-config-cluster1
      namespace: open-cluster-management
```

## Placement Auto-Install

Use placement mode only if you want the addon to be installed automatically based on cluster labels or clustersets.

### GPU placement

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
- a labeled cluster eventually gets `managedclusteraddon/flock-addon`

### All-cluster placement

```bash
# [Hub]
make deploy-auto-all
```

Check:

```bash
# [Hub]
kubectl -n open-cluster-management get placement flock-addon-all-placement -o yaml
kubectl -n cluster1 get managedclusteraddon flock-addon
```

Should see:

- the placement exists
- selected clusters eventually get `managedclusteraddon/flock-addon`

## Direct CLI Mapping

Old direct run:

```bash
python main.py \
  --task-address 0x47B0397C6ae306002788D093b29bcD2EDAd19924 \
  --dataset data/asr_sarawakmalay_whisper_format_client_ids.json \
  --hf-token $HF_TOKEN \
  --gpu
```

Addon mapping:

- `--task-address` maps to `deploymentConfig.blockchain.taskAddress`
- `--dataset` maps to `DATA_PATH`
- `--hf-token` comes from node `.env`
- `--gpu` maps to `deploymentConfig.runtime.useGpu=true`

Effective priority:

- CLI overrides
- environment variables
- YAML config defaults

## Validate Chart

```bash
# [Hub]
make verify
make test-chart
```

Check:

```bash
# [Hub]
make status
```

Should see:

- addon resources exist on the Hub
- managed cluster addon objects appear for enabled clusters

## Next Steps

- Use [Install FLock Addon](docs/install-flock-addon.md) for a clean first deployment
- Use [Troubleshooting](docs/troubleshooting.md) if the addon reaches Hub but not the managed cluster
- Use [Auto-Install by Placement](docs/auto-install-by-placement.md) only when you want automatic cluster selection
