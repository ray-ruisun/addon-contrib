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
- If your GPU nodes use taints or dedicated labels, set `agent.tolerations` and/or `agent.nodeSelector`

## Image Selection

Chart fallback image:

- `ghcr.io/flock-io/fl-alliance-client:<release-tag>`

Two common cases:

1. Public image repository:
   - recommended default path
   - no image pull secret required
   - usually deploy directly from `flock-addon`
2. Private image repository:
   - use when testing your own unpublished or restricted image
   - requires image publish first
   - requires `ghcr-pull` or another image pull secret on each managed cluster

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
IMAGE_OWNER='ray-ruisun' IMAGE_TAG='<git-sha-or-release-tag>' make deploy
```

Or:

```bash
# [Hub]
FLOCK_ALLIANCE_IMAGE='ghcr.io/ray-ruisun/fl-alliance-client:<git-sha-or-release-tag>' make deploy
```

Recommended explicit export form:

```bash
# [Hub]
export IMAGE_REGISTRY='ghcr.io'
export IMAGE_OWNER='ray-ruisun'
export IMAGE_NAME='fl-alliance-client'
export IMAGE_TAG='<git-sha-or-release-tag>'
export IMAGE_PULL_POLICY='Always'
export USE_GPU='true'
export GPU_RESOURCE_ENABLED='true'
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
- With `IMAGE_TAG='latest'`, keep `IMAGE_PULL_POLICY='Always'` so managed clusters do not reuse an old cached image.
- Prefer an immutable tag such as a git SHA or release tag for normal deployments.

If the image is private, also set:

```bash
# [Hub]
export IMAGE_PULL_SECRET='ghcr-pull'
```

If the selected registry is private, also configure `image.pullSecrets`.

## Publish To Deploy Flow

Recommended end-to-end flow for a custom image:

1. Publish `FL-Alliance-Client` image to GHCR.
2. Create `ghcr-pull` on each managed cluster if the package is private.
3. Deploy `flock-addon` from the Hub with matching image variables.
4. Verify the managed cluster Pod is pulling the expected image.

You only need the `FL-Alliance-Client` source repository if you want to build or publish your own image.
If you are using an already published image, you can skip the source checkout and deploy `flock-addon` directly from this repository.

Source repository:

- [FL-Alliance-Client](https://github.com/FLock-io/FL-Alliance-Client.git)

Clone it before local image build or manual image push:

```bash
# [Hub or image-build machine]
cd ~
git clone https://github.com/FLock-io/FL-Alliance-Client.git
cd FL-Alliance-Client
```

Example publish target:

```bash
# [FL-Alliance-Client workspace]
export IMAGE_SHA=$(git rev-parse --short=12 HEAD)
make image-build IMAGE_OWNER='ray-ruisun' IMAGE_TAG='latest' IMAGE_IMMUTABLE_TAG="$IMAGE_SHA"
make image-inspect IMAGE_OWNER='ray-ruisun' IMAGE_TAG='latest' IMAGE_IMMUTABLE_TAG="$IMAGE_SHA"
echo "$GHCR_PAT" | docker login ghcr.io -u "$GHCR_USER" --password-stdin
make image-push IMAGE_OWNER='ray-ruisun' IMAGE_TAG='latest' IMAGE_IMMUTABLE_TAG="$IMAGE_SHA"
```

Check:

```bash
# [FL-Alliance-Client workspace]
make image-inspect IMAGE_OWNER='ray-ruisun' IMAGE_TAG='latest' IMAGE_IMMUTABLE_TAG="$IMAGE_SHA"
```

Should see:

- local image exists for both `latest` and `$IMAGE_SHA`
- deploy the addon with `$IMAGE_SHA`, not `latest`

If you use the GitHub Actions workflow instead of local push, wait for the publish job to finish successfully before deploying the addon.

### Public Repository Path

Use this path when the image already exists publicly in GHCR, for example:

- `ghcr.io/flock-io/fl-alliance-client:<release-tag>`

How to operate:

1. Do not clone `FL-Alliance-Client` unless you want to rebuild the image.
2. Do not create `ghcr-pull`.
3. Deploy directly from `flock-addon`.

Example:

```bash
# [Hub]
unset IMAGE_PULL_SECRET
export IMAGE_OWNER='flock-io'
export IMAGE_TAG='<release-tag>'
make deploy-testnet TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
```

Check:

```bash
# [Hub]
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config -o yaml | rg -n "FLOCK_ALLIANCE_IMAGE|value"
```

Should see:

- `FLOCK_ALLIANCE_IMAGE` points to the public image
- no extra pull-secret configuration is required

### Private Repository Path

Use this path when the image is private, for example:

- `ghcr.io/ray-ruisun/fl-alliance-client:<git-sha-or-release-tag>`

How to operate:

1. Clone `FL-Alliance-Client` if you need to build or push the image yourself.
2. Publish the image to your private GHCR namespace.
3. Create `ghcr-pull` on each managed cluster.
4. Deploy `flock-addon` with `IMAGE_PULL_SECRET`.

Example:

```bash
# [Hub]
export IMAGE_OWNER='ray-ruisun'
export IMAGE_TAG='<git-sha-or-release-tag>'
export IMAGE_PULL_SECRET='ghcr-pull'
export IMAGE_PULL_POLICY='Always'
make deploy-testnet TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
```

Check:

```bash
# [Managed Cluster context]
kubectl -n flock-system get secret ghcr-pull
kubectl -n flock-system get deploy flock-agent -o yaml | rg -n "imagePullSecrets|ghcr-pull"
```

Should see:

- `ghcr-pull` exists
- `flock-agent` references `imagePullSecrets`

## Quick Start: Testnet Mode

This is the default deployment mode.

Behavior:

- `TASK_ADDRESS` must be passed from the Hub at deploy time
- `BLOCKCHAIN_RPC` and `TOKEN_ADDRESS` are read from each cluster node `.env`
- `PRIVATE_KEY` and `HF_TOKEN` are read from each cluster node `.env`
- GPU runtime is enabled by default through `deploymentConfig.runtime.useGpu=true`
- deploy targets also request GPU resources by default (`nvidia.com/gpu=1`)

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
export USE_GPU='true'
export GPU_RESOURCE_ENABLED='true'
make deploy-testnet TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
```

If you want a different image owner:

```bash
# [Hub]
cd flock-addon
export IMAGE_SHA=$(git rev-parse --short=12 HEAD)
USE_GPU='true' GPU_RESOURCE_ENABLED='true' IMAGE_OWNER='ray-ruisun' IMAGE_TAG="$IMAGE_SHA" make deploy-testnet TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
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
kubectl -n flock-system get pod -l app.kubernetes.io/name=flock-addon -o jsonpath='{range .items[*]}{.metadata.name}{"\trequest="}{.spec.containers[0].resources.requests.nvidia\.com/gpu}{"\tlimit="}{.spec.containers[0].resources.limits.nvidia\.com/gpu}{"\n"}{end}'
kubectl get node -o custom-columns=NAME:.metadata.name,GPU_ALLOCATABLE:.status.allocatable.nvidia\\.com/gpu
```

Should see:

- `deployment/flock-agent` exists
- Pod becomes `Running`
- logs show `FLockAlliance` startup rather than image pull or crash errors
- Pod resources show `request=1` and `limit=1` for `nvidia.com/gpu`
- at least one node shows non-empty `GPU_ALLOCATABLE`
- pod startup logs include either `NVIDIA device files detected` or `No NVIDIA device files detected in container`

If you intentionally want CPU mode:

```bash
# [Hub]
USE_GPU='false' GPU_RESOURCE_ENABLED='false' make deploy-testnet TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
```

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

If the image is wrong or still looks old after republishing the same tag, redeploy with an explicit override:

```bash
# [Hub]
IMAGE_OWNER='ray-ruisun' IMAGE_TAG='<git-sha-or-release-tag>' IMAGE_PULL_POLICY='Always' make deploy-testnet TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
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
- the Pod pulls the newest `latest` image instead of reusing a cached local copy

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
  --set image.tag="$IMAGE_SHA" \
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

## GPU Mapping Troubleshooting

If training is unexpectedly slow, verify the addon Pod is actually bound to GPU resources.

```bash
# [Managed Cluster context]
kubectl -n flock-system get pod -l app.kubernetes.io/name=flock-addon -o wide
kubectl -n flock-system describe pod -l app.kubernetes.io/name=flock-addon | rg -n "nvidia.com/gpu|Node:|Warning|FailedScheduling"
kubectl get node -o custom-columns=NAME:.metadata.name,GPU_ALLOCATABLE:.status.allocatable.nvidia\\.com/gpu
kubectl get ds -A | rg -i "nvidia|gpu|device-plugin"
kubectl -n flock-system logs deploy/flock-agent -c flock-alliance-client --tail=80 | rg -n "NVIDIA device files|nvidia-smi|No NVIDIA device files"
```

Should see:

- Pod is scheduled on a node with non-zero `GPU_ALLOCATABLE`
- container resources include `nvidia.com/gpu`
- no `FailedScheduling` due to missing GPU resources
- GPU device plugin DaemonSet exists and is Ready
- startup logs confirm whether `/dev/nvidia*` is visible inside the container

Then check FLocKit subprocess logs for device selection:

```bash
# [Managed Cluster context]
POD=$(kubectl -n flock-system get pod -l app.kubernetes.io/component=agent -o jsonpath='{.items[0].metadata.name}')
kubectl -n flock-system exec "$POD" -c flock-alliance-client -- sh -lc 'f=$(ls -1t /app/output/task_outputs/process_*.log | head -n1); echo "LOG=$f"; rg -n "CUDA is available|CUDA not available|Using device/backend|device=" "$f" || true'
```

Should see one of:

- GPU path: `CUDA is available` and CUDA device/backend lines
- CPU fallback: `CUDA not available` (then cluster/node GPU plugin or resource request is still not effective)

If your cluster dedicates GPU nodes with labels or taints, deploy with node placement hints, for example:

```bash
# [Hub]
helm upgrade --install flock-addon charts/flock-addon \
  --set agent.nodeSelector.gpu=true \
  --set 'agent.tolerations[0].key=nvidia.com/gpu' \
  --set 'agent.tolerations[0].operator=Exists' \
  --set 'agent.tolerations[0].effect=NoSchedule'
```

## Direct Client / FLocKit Log Troubleshooting

When using OCM + direct client mode, `FLocKit` is started as a subprocess inside
the same `flock-alliance-client` container. There is no separate `FLocKit` Pod.

If logs stop at:

- `[Proposer R*] Step 3/5: Training local model...`

collect both client logs and subprocess logs from the same Pod.

### 1) Find the running Pod

```bash
# [Managed Cluster context]
POD=$(kubectl -n flock-system get pod -l app.kubernetes.io/component=agent -o jsonpath='{.items[0].metadata.name}')
echo "$POD"
```

Check:

```bash
# [Managed Cluster context]
kubectl -n flock-system get pod "$POD"
```

Should see:

- a valid Pod name in `$POD`
- Pod status is `Running` (or `CrashLoopBackOff` if it is repeatedly failing)

### 2) Read FL-Alliance-Client logs

```bash
# [Managed Cluster context]
kubectl -n flock-system logs "$POD" -c flock-alliance-client --tail=200
```

Check:

```bash
# [Managed Cluster context]
kubectl -n flock-system logs "$POD" -c flock-alliance-client --tail=200 | rg -n "Model process logs:|Step 3/5|timed out|Process crashed"
```

Should see:

- startup lines from `FLockAlliance`
- `Model process logs: /app/output/task_outputs/process_*.log`
- timeout/crash hints if subprocess calls fail

### 3) Read FLocKit subprocess logs inside the container

```bash
# [Managed Cluster context]
kubectl -n flock-system exec "$POD" -c flock-alliance-client -- sh -lc 'ls -lt /app/output/task_outputs | head -n 20'
```

Check:

```bash
# [Managed Cluster context]
kubectl -n flock-system exec "$POD" -c flock-alliance-client -- sh -lc 'f=$(ls -1t /app/output/task_outputs/process_*.log | head -n1); echo "LOG=$f"; tail -n 200 "$f"'
```

Should see:

- latest `process_*.log` path
- traceback or template/runtime error from `FLocKit` (if training failed)

### 4) Live follow subprocess logs

```bash
# [Managed Cluster context]
kubectl -n flock-system exec "$POD" -c flock-alliance-client -- sh -lc 'f=$(ls -1t /app/output/task_outputs/process_*.log | head -n1); tail -f "$f"'
```

Should see:

- continuous training/evaluation output from `FLocKit`
- if stalled, no new lines for a long period; if crashed, traceback appears

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
