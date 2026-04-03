# Install FLock Addon

This guide walks through the recommended first deployment for `flock-addon`: `Local chain + local S3-compatible`, with one `flock-agent` running on each enabled managed cluster.

Before using this guide, complete [Prepare Multi-Cluster Environment](prepare-multicluster-environment.md). In particular, make sure:

- the hub and managed clusters are separate Kubernetes clusters
- OCM registration is complete and `ManagedCluster` objects are `Joined=True` and `Available=True`
- a simple `ManifestWork` can already reach the managed clusters
- single-node clusters have their control-plane taints removed if they need to run workloads on the only node

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
  - `.env` and local data files used by `FLockAlliance`

## Prerequisites

- OCM hub and managed clusters are already available and healthy
- `kubectl`, `helm`, and `make` are installed on the hub
- Every node that may run the addon Pod has a shared host path, usually `/data/flock-client`
- This repository is checked out on the hub machine:

```bash
cd /path/to/addon-contrib/flock-addon
```

For the recommended default path, also prepare:

- a checkout of `FL-Alliance-Client` on the hub machine
- a local model archive such as `/absolute/path/to/model.tar.gz`
- a hub IP or hostname reachable from managed clusters for `RPC_HOST`

If you need a different deployment path, use [Deployment Modes](deployment-modes.md). If you need a custom or private image, use [Image Management](image-management.md) before enabling the addon.

If you are following older testing notes or screenshots, the old `make deploy` flow maps conceptually to the hub-side addon deploy step, but the current `Makefile` uses explicit mode-specific commands such as `make deploy-local-chain-s3-compatible`.

## Step 1: Prepare the Node Path

Run on every managed cluster node that may host the addon Pod.

```bash
# [Each Managed Cluster Node]
sudo mkdir -p /data/flock-client
sudo chmod 755 /data
sudo chown -R <login-user>:<login-group> /data/flock-client
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

If your workflow depends on node-local input files, copy them into `/data/flock-client` now. This directory is mounted into the container at `/data`.

## Step 2: Create the Node `.env`

Create this file on every managed cluster node:

```text
/data/flock-client/.env
```

Recommended `.env` for `Local chain + local S3-compatible`:

```dotenv
PRIVATE_KEY=<private-key>
HF_TOKEN=<hf-token>
```

Ignore any secrets shown in historical testing notes. The important part is the variable layout, not the sample values.

Check:

```bash
# [Each Managed Cluster Node]
ls -l /data/flock-client/.env
sed -n '1,20p' /data/flock-client/.env
```

Should see:

- `.env` exists at `/data/flock-client/.env`
- `PRIVATE_KEY` and `HF_TOKEN` are present

In the recommended default mode, blockchain RPC, task address, token address, and S3-compatible storage settings are pushed from the hub. Node `.env` only needs node-local secrets.

## Step 3: Deploy the Addon Definition on the Hub

Deploy the shared addon definition from the hub using the recommended self-contained mode:

```bash
# [Hub]
cd flock-addon
make deploy-local-chain-s3-compatible \
  FL_ALLIANCE_CLIENT_DIR=/path/to/FL-Alliance-Client \
  MODEL_ARCHIVE=/absolute/path/to/model.tar.gz \
  RPC_HOST=<hub-ip> \
  DOCKER='sudo docker' \
  S3_COMPAT_DATA_DIR='<local-minio-data-dir>'
```

If your user can already access Docker without `sudo`, you can omit `DOCKER='sudo docker'`.

If you prefer the default MinIO data directory, create it first:

```bash
sudo mkdir -p /srv/flock-minio/data
sudo chown -R "$USER":"$(id -gn)" /srv/flock-minio
```

Optional image overrides:

```bash
# [Hub]
export IMAGE_REGISTRY='ghcr.io'
export IMAGE_OWNER='<image-owner>'
export IMAGE_NAME='fl-alliance-client'
export IMAGE_TAG='<git-sha-or-release-tag>'
export IMAGE_PULL_POLICY='Always'
export FLOCK_ALLIANCE_IMAGE="${IMAGE_REGISTRY}/${IMAGE_OWNER}/${IMAGE_NAME}:${IMAGE_TAG}"
```

If the selected image is private, create the managed-cluster pull secret first and set `IMAGE_PULL_SECRET`. The full flow is in [Image Management](image-management.md).

If the addon definition exists on the hub but workloads never reach managed clusters, stop here and validate the OCM pipeline with [Prepare Multi-Cluster Environment](prepare-multicluster-environment.md).

Check:

```bash
# [Hub]
kubectl get clustermanagementaddon flock-addon
kubectl -n open-cluster-management get addontemplate flock-addon
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config -o yaml | rg -n "TASK_ADDRESS|BLOCKCHAIN_RPC|S3_COMPAT_ENDPOINT_URL|FLOCK_ALLIANCE_IMAGE|value"
```

Should see:

- `clustermanagementaddon/flock-addon` exists
- `addontemplate/flock-addon` exists
- `addontemplate/flock-addon-gpu` exists
- `addondeploymentconfig/flock-addon-config` exists
- `addondeploymentconfig/flock-addon-gpu-config` exists
- `TASK_ADDRESS` matches the hub-generated value
- `BLOCKCHAIN_RPC` points to the hub-hosted local chain
- `S3_COMPAT_ENDPOINT_URL` points to the hub-hosted local S3-compatible service

## Step 4: Enable the Addon on a Managed Cluster

GPU/CPU template selection follows the hub-side `managedcluster` label `gpu=true`.

```bash
# [Hub]
make enable-addon CLUSTER=<cluster-name>
```

Check:

```bash
# [Hub]
kubectl -n <cluster-name> get managedclusteraddon flock-addon -o yaml
kubectl -n <cluster-name> get manifestwork
```

Should see:

- `managedclusteraddon/flock-addon` exists
- `spec.configs` selects the GPU template/config on `gpu=true` clusters
- `spec.configs` selects the CPU template/config on other clusters
- a `ManifestWork` appears

To enable multiple clusters, repeat the same command:

```bash
# [Hub]
make enable-addon CLUSTER=<cluster-a>
make enable-addon CLUSTER=<cluster-b>
make enable-addon CLUSTER=<cluster-c>
```

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
- the Pod becomes `Running`
- logs show `FLockAlliance` startup
- logs include the local chain and S3-compatible runtime path instead of missing RPC or storage errors
- on `gpu=true` clusters, Pod resources show `request=1` and `limit=1` for `nvidia.com/gpu`
- on CPU clusters, the GPU request fields are empty and the Pod still runs

## Cleanup

```bash
# [Hub]
make disable-addon CLUSTER=<cluster-name>
make undeploy
```

## Next Steps

- Use [Deployment Modes](deployment-modes.md) for testnet or external-S3 workflows
- Use [Image Management](image-management.md) for public/private registry setups and custom image publishing
- Use [Configuration and Overrides](configuration-and-overrides.md) for task updates, path rules, and per-cluster customization
- Use [Troubleshooting](troubleshooting.md) if the rollout reaches the hub but not the managed cluster
