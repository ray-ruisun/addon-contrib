# FLock Addon

FLock addon deploys **FLockAlliance** and **FLocKit** together on managed clusters as one logical unit.

- `FLockAlliance` (`fl-alliance-client` image): blockchain/protocol orchestration (participant loop)
- `FLocKit`: model API sidecar (`POST /call`)

The addon uses OCM AddOnTemplate and supports both manual enablement and placement-based auto-install.

## Architecture

- One Deployment per managed cluster (`flock-agent`)
- Two containers in the same Pod:
  - `flock-alliance-client` (runtime mode `redhat_ocm`)
  - `flockit` (serves API at `http://127.0.0.1:5000`)
- Shared volume mounted at `/data` (`emptyDir` by default, optional PVC/hostPath)
- Both containers can load an env file from the mounted volume (default: `/data/.env`)

## Prerequisites

- OCM hub + managed clusters
- A blockchain private key secret on managed clusters
- Valid `rpc`, `tokenAddress`, `taskAddress` for your chain mode:
  - local chain: point `rpc` to your local endpoint and pass local deployed addresses
  - testnet: point `rpc` + addresses to testnet contracts

Create secret in each managed cluster namespace (default install namespace: `flock-system`):

```bash
kubectl -n flock-system create secret generic flock-alliance-secret \
  --from-literal=CLIENT_PRIVATE_KEY='0x...' \
  --from-literal=HF_TOKEN='hf_...'
```

## Deploy

```bash
cd flock-addon
make deploy
```

Override runtime chain/storage fields at deploy time:

```bash
helm upgrade --install flock-addon charts/flock-addon \
  --set deploymentConfig.blockchain.rpc='http://10.0.0.10:8545' \
  --set deploymentConfig.blockchain.tokenAddress='0x...' \
  --set deploymentConfig.blockchain.taskAddress='0x...' \
  --set deploymentConfig.storage.backend='local' \
  --set deploymentConfig.storage.localSharedDir='/data/shared'
```

For `deploymentConfig.storage.backend=local`, all participants must see the same
shared filesystem path (for example via NFS-backed PVC mounted in each cluster).
Set `agent.dataVolume.existingClaim=<rwx-claim>` (or `hostPath` for single-node dev).

## Use Mounted `.env` (Recommended)

The addon supports loading runtime parameters from a mounted env file.
By default, both containers load `/data/.env` if it exists.

- `flock-alliance-client`: controlled by `deploymentConfig.runtime.flockAllianceEnvFile`
- `flockit`: controlled by `deploymentConfig.flockit.envFile`

Default values are already set to `/data/.env`.

Example with hostPath:

```bash
helm upgrade --install flock-addon charts/flock-addon \
  --set agent.dataVolume.hostPath='/opt/flock-shared' \
  --set deploymentConfig.runtime.flockAllianceEnvFile='/data/.env' \
  --set deploymentConfig.flockit.envFile='/data/.env'
```

Then place your env file on each managed cluster node at:

```text
/opt/flock-shared/.env
```

Example `.env`:

```dotenv
# FLockAlliance
PRIVATE_KEY=0x...
HF_TOKEN=hf_...
MODEL_API_URL=http://127.0.0.1:5000
BLOCKCHAIN_RPC=https://sepolia.base.org
TOKEN_ADDRESS=0x...
TASK_ADDRESS=0x...
STAKE=0
STORAGE_BACKEND=s3
LOCAL_STORAGE_DIR=/data/shared
USE_GPU=false
NO_INCENTIVE=false
NUM_PARTICIPANTS=1
DATA_PATH=/data

# FLocKit
FLOCKIT_CONF=templates/llm_finetuning/configs/addon_default.yaml
FLOCKIT_PORT=5000
FLOCKIT_DATA_PATH=/data
FLOCKIT_DATA_SOURCE=
FLOCKIT_DATA_INDICES_PATH=
FLOCKIT_OVERRIDES=
```

Priority notes:

- AddOnDeploymentConfig provides default environment variables.
- If the env file exists, it is sourced at container startup and can override those defaults.
- `FLockAlliance` then applies its own priority: CLI overrides > environment > YAML config.
- In this addon, CLI overrides are built from environment values after env-file loading, so env-file values are reflected in runtime.

## Per-Cluster Config Override

If one cluster needs different runtime variables, create a dedicated
`AddOnDeploymentConfig` on the hub and reference it from that cluster's
`ManagedClusterAddOn`.

Example dedicated config:

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
    - name: FLOCKIT_ENV_FILE
      value: /data/.env
    - name: MODEL_API_URL
      value: http://127.0.0.1:5000
    - name: DATA_PATH
      value: /data
```

Reference it from `ManagedClusterAddOn`:

```yaml
apiVersion: addon.open-cluster-management.io/v1alpha1
kind: ManagedClusterAddOn
metadata:
  name: flock-addon
  namespace: cluster1
spec:
  installNamespace: flock-system
  configs:
    - group: addon.open-cluster-management.io
      resource: addondeploymentconfigs
      name: flock-addon-config-cluster1
      namespace: open-cluster-management
```

## Enable Addon On A Cluster

```bash
make enable-addon CLUSTER=cluster1
```

## Placement Auto-Install

```bash
make deploy-auto-gpu
# label cluster
kubectl label managedcluster cluster1 gpu=true
```

or:

```bash
make deploy-auto-all
```

## FLocKit Flexibility

FLocKit sidecar is controlled by variables in `AddOnDeploymentConfig`:

- `FLOCKIT_CONF`: base YAML template path in image
- `FLOCKIT_OVERRIDES`: comma/newline `key=value` overrides
- `FLOCKIT_PORT`, `FLOCKIT_DATA_PATH`, `FLOCKIT_DATA_SOURCE`, `FLOCKIT_DATA_INDICES_PATH`
- `FLOCKIT_ENV_FILE`: optional env file path loaded before `FLocKit` starts

This allows frequent model/template changes without modifying FLockAlliance code.

## OCM Validation Checklist

After enabling on a managed cluster:

```bash
kubectl get clustermanagementaddon flock-addon -o yaml
kubectl -n cluster1 get managedclusteraddon flock-addon -o yaml
kubectl -n flock-system get deploy,pod
kubectl -n flock-system logs deploy/flock-agent -c flock-alliance-client --tail=100
kubectl -n flock-system logs deploy/flock-agent -c flockit --tail=100
```

What to verify:

- `ManagedClusterAddOn` condition shows install/available progressing normally.
- `flock-agent` Pod has both containers running (`flock-alliance-client`, `flockit`).
- `flock-alliance-client` log shows `runtime.mode=redhat_ocm`.
- `flockit` is listening on `${FLOCKIT_PORT}` (default `5000`).
- If `.env` is mounted, expected values appear in startup behavior (RPC, task address, data path, template config).

## Validate Chart

```bash
make verify
make test-chart
```
