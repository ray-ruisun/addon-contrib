# Install FLock Addon

This guide installs the FLock addon on Open Cluster Management (OCM) and deploys `FLockAlliance` to managed clusters.

## What Gets Deployed

- Hub cluster:
  - `ClusterManagementAddOn`
  - `AddOnTemplate`
  - `AddOnDeploymentConfig`
- Managed cluster:
  - namespace `flock-system`
  - Deployment `flock-agent`
  - container `flock-alliance-client`
- Managed cluster node:
  - mounted host path, usually `/data/flock-client`
  - `.env` and local data files

## Prerequisites

- OCM Hub and managed clusters are already available
- `kubectl`, `helm`, and `make` are installed on the Hub
- every node that may run the addon Pod has a shared host path, usually `/data/flock-client`

## Image Publish Before Addon Deploy

If you want to deploy a custom image, publish it first from `FL-Alliance-Client`.

This step is only needed when you are building or publishing your own image.
If you will deploy an image that already exists in GHCR, you do not need to clone the source repository first.

Source repository:

- [FL-Alliance-Client](https://github.com/FLock-io/FL-Alliance-Client.git)

Clone the source repository before local build or push:

```bash
# [Hub or image-build machine]
cd ~
git clone https://github.com/FLock-io/FL-Alliance-Client.git
cd FL-Alliance-Client
```

Local publish example:

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

- both `latest` and `$IMAGE_SHA` exist locally before push
- addon deployment should use `$IMAGE_SHA`

If you use GitHub Actions publishing, wait for the workflow to finish before running `flock-addon` deployment.

## Public vs Private Image Repository

### Public Repository

Use this when the image is already published publicly, for example:

- `ghcr.io/flock-io/fl-alliance-client:<release-tag>`

How to operate:

- you do not need to clone `FL-Alliance-Client`
- you do not need to create `ghcr-pull`
- you can deploy `flock-addon` directly from the Hub

Example:

```bash
# [Hub]
unset IMAGE_PULL_SECRET
export IMAGE_OWNER='flock-io'
export IMAGE_TAG='<release-tag>'
```

### Private Repository

Use this when the image is private, for example:

- `ghcr.io/ray-ruisun/fl-alliance-client:<git-sha-or-release-tag>`

How to operate:

- clone `FL-Alliance-Client` if you need local build or push
- publish the image before addon deployment
- create `ghcr-pull` on every managed cluster
- export `IMAGE_PULL_SECRET='ghcr-pull'` on the Hub before deploy

Example:

```bash
# [Hub]
export IMAGE_OWNER='ray-ruisun'
export IMAGE_TAG='<git-sha-or-release-tag>'
export IMAGE_PULL_SECRET='ghcr-pull'
```

## Step 1: Prepare the Node Path

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

## Step 2: Create the Node `.env`

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

## Step 3: Deploy the Addon Definition on the Hub

Default testnet mode:

- `TASK_ADDRESS` is passed from the Hub
- `STORAGE_BACKEND` and `NO_INCENTIVE` are controlled from the Hub deploy config
- `BLOCKCHAIN_RPC` and `TOKEN_ADDRESS` are read from node `.env`
- `PRIVATE_KEY` and `HF_TOKEN` are read from node `.env`

Optional image variables:

```bash
# [Hub]
export IMAGE_REGISTRY='ghcr.io'
export IMAGE_OWNER='ray-ruisun'
export IMAGE_NAME='fl-alliance-client'
export IMAGE_SHA=$(git rev-parse --short=12 HEAD)
export IMAGE_TAG="$IMAGE_SHA"
export FLOCK_ALLIANCE_IMAGE="${IMAGE_REGISTRY}/${IMAGE_OWNER}/${IMAGE_NAME}:${IMAGE_TAG}"
```

If the registry is private:

```bash
# [Hub]
export IMAGE_PULL_SECRET='ghcr-pull'
```

If the package is private, create the pull secret on each managed cluster before enabling the addon:

```bash
# [Managed Cluster context]
kubectl -n flock-system create secret docker-registry ghcr-pull \
  --docker-server=ghcr.io \
  --docker-username="$GHCR_USER" \
  --docker-password="$GHCR_PAT" \
  --dry-run=client -o yaml | kubectl apply -f -
```

Check:

```bash
# [Managed Cluster context]
kubectl -n flock-system get secret ghcr-pull
```

Should see:

- secret `ghcr-pull` exists before addon rollout

```bash
# [Hub]
cd flock-addon
make deploy-testnet TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
```

If you need a different image owner:

```bash
# [Hub]
cd flock-addon
IMAGE_OWNER='ray-ruisun' IMAGE_TAG="$IMAGE_SHA" make deploy-testnet TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
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
- `addontemplate/flock-addon-gpu` exists
- `addondeploymentconfig/flock-addon-config` exists
- `addondeploymentconfig/flock-addon-gpu-config` exists
- `TASK_ADDRESS` matches the value you passed
- `FLOCK_ALLIANCE_IMAGE` matches the image you expect

Notes:

- `AddOnDeploymentConfig.customizedVariables` is still how runtime values are passed into the addon Pod.
- Old Helm value paths such as `placement.all.config.useGpu` were removed during cleanup.
- GPU resource scheduling is now controlled by the selected `AddOnTemplate`, while runtime flags such as `USE_GPU` still come from `AddOnDeploymentConfig`.
- `TASK_ADDRESS`, `USE_GPU`, `STORAGE_BACKEND`, and `NO_INCENTIVE` stay authoritative from OCM.
- when `STORAGE_BACKEND=local`, `BLOCKCHAIN_RPC`, `TOKEN_ADDRESS`, and `LOCAL_STORAGE_DIR` also stay authoritative from OCM when non-empty.
- `NUM_PARTICIPANTS` is forced from OCM only when `STORAGE_BACKEND=local`; it is not forced in testnet/S3 mode.

## Step 4: Enable the Addon on a Managed Cluster

GPU/CPU template selection follows the Hub-side `managedcluster` label `gpu=true`.

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
- `spec.configs` selects the GPU template/config on `gpu=true` clusters
- `spec.configs` selects the CPU template/config on other clusters
- a `ManifestWork` appears

## Step 5: Verify Runtime on the Managed Cluster

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
- logs show `FLockAlliance` startup
- the Pod pulls the image matching `FLOCK_ALLIANCE_IMAGE`
- on `gpu=true` clusters, Pod resources show `request=1` and `limit=1` for `nvidia.com/gpu`
- on CPU clusters, the GPU request fields are empty and the Pod still runs
- at least one GPU-enabled node shows non-empty `GPU_ALLOCATABLE`

## Local Chain Mode

Use local chain mode only if the blockchain endpoint is reachable from the addon Pod.

Important:

- do not use `127.0.0.1` unless the chain runs in the same Pod
- prefer a node IP or Kubernetes Service DNS reachable from `flock-agent`
- `storage.backend=local` requires a real shared filesystem for `/data/shared`
- do not make the whole `/data/flock-client` directory shared across clusters, because `.env` is per-node and should keep a different `PRIVATE_KEY` on each managed cluster

Recommended topology when the Hub hosts both Anvil and the shared storage:

1. On the Hub or chain host, run `make chain MODEL_DEFINITION_HASH=<hash>` in `FL-Alliance-Client`.
2. Export one shared directory from the Hub, for example `/srv/flock-shared`.
3. On each managed cluster node:
   - keep `/data/flock-client/.env` as a node-local file
   - mount the Hub export at `/data/flock-client/shared`
   - copy the Hub-generated `data/contracts.json` to `/data/flock-client/contracts.json`, or pass `TOKEN_ADDRESS` and `TASK_ADDRESS` from the Hub deploy command

If you want the Hub to start the local chain as part of addon deployment, use the
one-command wrapper:

```bash
# [Hub]
make deploy-local-stack \
  FL_ALLIANCE_CLIENT_DIR=/path/to/FL-Alliance-Client \
  MODEL_ARCHIVE=/path/to/model.tar.gz \
  RPC_HOST=<hub-ip> \
  HUB_SHARED_MODELS_DIR=/srv/flock-shared/models
```

This wrapper starts `make chain` in `FL-Alliance-Client`, reads the deployed
contract addresses, and then deploys `flock-addon` in local mode. It still does
not mount NFS/SMB for you on the managed nodes, and it does not create each
node's `.env` automatically.

When you use this wrapper, managed nodes do not need a local `contracts.json`
copy because the Hub passes the contract addresses directly.

Example node `.env`:

```dotenv
PRIVATE_KEY=0x...
HF_TOKEN=hf_...
BLOCKCHAIN_RPC=http://<hub-ip-or-service>:8545
```

Place the model archive into the shared storage before enabling the addon:

```bash
# [Hub or chain host]
mkdir -p /srv/flock-shared/models
cp model.tar.gz /srv/flock-shared/models/<MODEL_HASH>
```

If you are not passing contract addresses from the Hub, also copy the generated contract
metadata to each managed node as `/data/flock-client/contracts.json`:

```bash
# [Hub or chain host]
scp ./data/contracts.json \
  ubuntu@<managed-node>:/data/flock-client/contracts.json
```

Deploy:

```bash
# [Hub]
make deploy-local-chain RPC='http://<hub-ip-or-service>:8545'
```

If you want the Hub to push the contract addresses directly instead of copying
`contracts.json` to every managed node:

```bash
# [Hub]
make deploy-local-chain \
  RPC='http://<hub-ip-or-service>:8545' \
  TOKEN_ADDRESS='0x<token-address>' \
  TASK_ADDRESS='0x<task-address>'
```

Check:

```bash
# [Hub]
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config -o yaml | rg -n "BLOCKCHAIN_RPC|TOKEN_ADDRESS|TASK_ADDRESS|STORAGE_BACKEND|LOCAL_STORAGE_DIR|value"
```

Should see:

- `BLOCKCHAIN_RPC` points to the Hub-hosted chain
- `STORAGE_BACKEND` is `local`
- `LOCAL_STORAGE_DIR` is `/data/shared`
- `TOKEN_ADDRESS` / `TASK_ADDRESS` are present only if you passed them from the Hub
