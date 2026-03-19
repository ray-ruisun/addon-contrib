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

## Step 3: Deploy the Addon Definition on the Hub

Default testnet mode:

- `TASK_ADDRESS` is passed from the Hub
- `BLOCKCHAIN_RPC` and `TOKEN_ADDRESS` are read from node `.env`
- `PRIVATE_KEY` and `HF_TOKEN` are read from node `.env`

Optional image variables:

```bash
# [Hub]
export IMAGE_REGISTRY='ghcr.io'
export IMAGE_OWNER='ray-ruisun'
export IMAGE_NAME='fl-alliance-client'
export IMAGE_TAG='v0.1.0'
export FLOCK_ALLIANCE_IMAGE="${IMAGE_REGISTRY}/${IMAGE_OWNER}/${IMAGE_NAME}:${IMAGE_TAG}"
```

```bash
# [Hub]
cd flock-addon
make deploy-testnet TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
```

If you need a different image owner:

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

## Step 4: Enable the Addon on a Managed Cluster

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
- a `ManifestWork` appears

## Step 5: Verify Runtime on the Managed Cluster

```bash
# [Managed Cluster context]
kubectl -n flock-system get deploy,pod
kubectl -n flock-system logs deploy/flock-agent -c flock-alliance-client --tail=100
```

Should see:

- `deployment/flock-agent` exists
- Pod becomes `Running`
- logs show `FLockAlliance` startup

## Local Chain Mode

Use local chain mode only if the blockchain endpoint is reachable from the addon Pod.

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

Deploy:

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
