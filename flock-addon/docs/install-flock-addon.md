# Install FLock Addon

This guide installs the FLock addon on Open Cluster Management (OCM) and deploys `FLockAlliance` to managed clusters.

## Primary Deployment Modes

The only supported deployment modes are:

1. `make deploy-testnet`
   - testnet blockchain
   - original signer-based `s3` storage backend
   - Hub does not push `BLOCKCHAIN_RPC`
2. `make deploy-local-chain-s3`
   - Hub automatically starts the local chain
   - original signer-based `s3` storage backend
   - Hub pushes the local-chain `BLOCKCHAIN_RPC`
   - you upload the model archive first and pass the existing `MODEL_HASH` to the Hub deploy command
3. `make deploy-local-chain-s3-compatible`
   - Hub automatically starts the local chain
   - Hub automatically starts local S3-compatible storage
   - Hub uploads `MODEL_ARCHIVE` into object storage using the SHA256 as the object key
   - Hub pushes the local-chain `BLOCKCHAIN_RPC`

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

Mode-specific `.env` requirements:

| Variable | `deploy-testnet` | `deploy-local-chain-s3` | `deploy-local-chain-s3-compatible` |
| --- | --- | --- | --- |
| `PRIVATE_KEY` | required | required | required |
| `HF_TOKEN` | required | required | required |
| `BLOCKCHAIN_RPC` | required, from node `.env` | not needed from `.env` | not needed from `.env` |
| `TOKEN_ADDRESS` | optional from `.env` when Hub leaves it empty | not needed from `.env` | not needed from `.env` |
| `S3_COMPAT_ENDPOINT_URL` | not used | not used | auto-pushed from Hub |
| `S3_COMPAT_BUCKET` | not used | not used | auto-pushed from Hub |
| `S3_COMPAT_ACCESS_KEY` | not used | not used | auto-pushed from Hub |
| `S3_COMPAT_SECRET_KEY` | not used | not used | auto-pushed from Hub |
| `S3_COMPAT_REGION` | not used | not used | auto-pushed from Hub |
| `S3_COMPAT_ADDRESSING_STYLE` | not used | not used | auto-pushed from Hub |
| `S3_COMPAT_VERIFY_SSL` | not used | not used | auto-pushed from Hub |

Examples:

`deploy-testnet`

```dotenv
PRIVATE_KEY=0x...
HF_TOKEN=hf_...
BLOCKCHAIN_RPC=https://sepolia.base.org
TOKEN_ADDRESS=0x...
```

`deploy-local-chain-s3`

```dotenv
PRIVATE_KEY=0x...
HF_TOKEN=hf_...
```

`deploy-local-chain-s3-compatible`

```dotenv
PRIVATE_KEY=0x...
HF_TOKEN=hf_...
```

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
- in testnet mode, `BLOCKCHAIN_RPC` comes from each node `.env`.
- when `STORAGE_BACKEND=local`, `BLOCKCHAIN_RPC`, `TOKEN_ADDRESS`, and `LOCAL_STORAGE_DIR` stay authoritative from OCM when non-empty.
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

## Local Chain + Original S3 Mode

Use this mode when:

- the Hub should auto-start the local chain
- storage should still use the original signer-based `s3` backend
- the Hub should push the local-chain `BLOCKCHAIN_RPC`
- you already uploaded `model.tar.gz` to the original/public S3 bucket and have the matching `MODEL_HASH`

Prepare the uploaded model first, for example:

```bash
# [FLocKit workspace]
python scripts/build_and_upload_s3.py --storage s3
```

That command prints the SHA256 hash. Use that value as `MODEL_HASH` below.

Deploy:

```bash
# [Hub]
make deploy-local-chain-s3 \
  FL_ALLIANCE_CLIENT_DIR=/path/to/FL-Alliance-Client \
  MODEL_HASH=<sha256> \
  RPC_HOST=<hub-ip>
```

If Docker on the Hub requires `sudo`, run the same command with
`DOCKER='sudo docker'`.

Important:

- wait for the command to finish by itself
- do not press `Ctrl+C` while `make chain` is still deploying contracts
- `anvil` is started in the background, but the Hub still waits for the one-shot `deployer` step to finish
- if you interrupt this step early, `data/contracts.json` may be missing or incomplete, and `TOKEN_ADDRESS` / `TASK_ADDRESS` will not be pushed to the addon

What it does:

- runs `make chain` in `FL-Alliance-Client`
- reads `TOKEN_ADDRESS` and `TASK_ADDRESS` from `data/contracts.json`
- deploys `storage.backend=s3`
- pushes `BLOCKCHAIN_RPC=http://<hub-ip>:8545`

Managed cluster node `.env` still needs:

```dotenv
PRIVATE_KEY=0x...
HF_TOKEN=hf_...
```

Check:

```bash
# [Hub]
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config -o yaml | rg -n "BLOCKCHAIN_RPC|TOKEN_ADDRESS|TASK_ADDRESS|STORAGE_BACKEND|value"

# [Hub]
test -f /path/to/FL-Alliance-Client/data/contracts.json && \
python3 - <<'PY'
import json
data = json.load(open('/path/to/FL-Alliance-Client/data/contracts.json'))
print("TOKEN_ADDRESS=", data["FlockToken"]["address"])
print("TASK_ADDRESS=", data["FlockTask"]["address"])
PY
```

Should see:

- `STORAGE_BACKEND` is `s3`
- `BLOCKCHAIN_RPC` points to `http://<hub-ip>:8545`
- `TASK_ADDRESS` matches the Hub-generated value
- the deployed task was created from the `MODEL_HASH` you passed to `make chain`

After enabling the addon on a managed cluster, verify runtime:

```bash
# [Hub]
make enable-addon CLUSTER=<cluster-name>

# [Managed Cluster context]
kubectl -n flock-system get deploy,pod
kubectl -n flock-system logs deploy/flock-agent -c flock-alliance-client --tail=100
```

Should see:

- `deployment/flock-agent` exists
- Pod becomes `Running`
- logs include the local-chain `TASK_ADDRESS`
- logs do not show missing `BLOCKCHAIN_RPC` errors

## Local Chain + Local S3-Compatible Storage Mode

Use this mode when the Hub should host both the local chain and a local
S3-compatible object store such as MinIO.

Important:

- this mode uses `storage.backend=nami`
- the Hub starts local S3-compatible storage automatically
- the Hub uploads `MODEL_ARCHIVE` automatically
- the Hub pushes the local-chain `BLOCKCHAIN_RPC`
- the Hub auto-pushes `S3_COMPAT_ENDPOINT_URL`, `S3_COMPAT_BUCKET`,
  `S3_COMPAT_ACCESS_KEY`, `S3_COMPAT_SECRET_KEY`, `S3_COMPAT_REGION`,
  `S3_COMPAT_ADDRESSING_STYLE`, and `S3_COMPAT_VERIFY_SSL`
- nodes only need their own local secrets such as `PRIVATE_KEY` and `HF_TOKEN`

The command auto-starts MinIO-compatible storage and auto-creates the bucket.
The manual startup below is only useful if you want to debug the Hub-side object
store separately.

Optional manual local S3-compatible service on the Hub:

```bash
# [Hub]
mkdir -p /srv/minio/data
docker run -d \
  --name minio \
  -p 9000:9000 \
  -p 9001:9001 \
  -e MINIO_ROOT_USER=minioadmin \
  -e MINIO_ROOT_PASSWORD=minioadmin \
  -v /srv/minio/data:/data \
  quay.io/minio/minio server /data --console-address ":9001"
```

Check:

```bash
# [Hub]
docker ps | rg minio
curl http://127.0.0.1:9000/minio/health/live
```

Should see:

- a running `minio` container
- an `OK` response from the health endpoint

Each managed cluster node `.env` should keep only node-local secrets:

```dotenv
PRIVATE_KEY=0x...
HF_TOKEN=hf_...
```

Deploy:

```bash
# [Hub]
make deploy-local-chain-s3-compatible \
  FL_ALLIANCE_CLIENT_DIR=/path/to/FL-Alliance-Client \
  MODEL_ARCHIVE=/path/to/model.tar.gz \
  RPC_HOST=<hub-ip>
```

If Docker on the Hub requires `sudo`, run the same command with
`DOCKER='sudo docker'`.

Important:

- wait for the command to finish by itself
- do not press `Ctrl+C` while `make chain` is still deploying contracts
- this mode also waits for the local MinIO upload and the one-shot `deployer` step before Helm deploy starts
- if you interrupt this step early, `data/contracts.json` may be missing or incomplete, and the addon will not get valid `TOKEN_ADDRESS` / `TASK_ADDRESS`

Check:

```bash
# [Hub]
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config -o yaml | rg -n "STORAGE_BACKEND|TASK_ADDRESS|value"
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config -o yaml | rg -n "S3_COMPAT_ENDPOINT_URL|S3_COMPAT_BUCKET|S3_COMPAT_ACCESS_KEY|S3_COMPAT_REGION|value"

# [Hub]
docker ps | rg 'flock-minio|minio'
curl http://127.0.0.1:9000/minio/health/live

# [Hub]
python3 - <<'PY'
import json
data = json.load(open('/path/to/FL-Alliance-Client/data/contracts.json'))
print("TOKEN_ADDRESS=", data["FlockToken"]["address"])
print("TASK_ADDRESS=", data["FlockTask"]["address"])
PY

# [Managed Cluster context]
kubectl -n flock-system logs deploy/flock-agent -c flock-alliance-client --tail=80 | rg -n "S3-compatible|storage backend|nami"
POD=$(kubectl -n flock-system get pod -l app.kubernetes.io/component=agent -o jsonpath='{.items[0].metadata.name}')
kubectl -n flock-system exec "$POD" -c flock-alliance-client -- sh -lc 'printenv | rg "S3_COMPAT_|BLOCKCHAIN_RPC|TASK_ADDRESS|TOKEN_ADDRESS"'
```

Should see:

- `STORAGE_BACKEND` is `nami`
- `BLOCKCHAIN_RPC` points to `http://<hub-ip>:8545`
- `TASK_ADDRESS` matches the Hub-generated value
- `S3_COMPAT_ENDPOINT_URL` points to `http://<hub-ip>:9000`
- client logs include `Using direct S3-compatible storage backend`
- Hub-side MinIO container is running and health returns `OK`
- the Pod environment contains `S3_COMPAT_ENDPOINT_URL`, `S3_COMPAT_BUCKET`, and the Hub-pushed local-chain addresses
