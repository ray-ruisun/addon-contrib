# FLock Addon

FLock addon deploys **FLockAlliance** on managed clusters as a **direct client** workload.

- Runtime mode is fixed to `local` (direct Python process execution)
- `FLocKit` is **not** deployed as an addon sidecar

The addon uses OCM `ClusterManagementAddOn` + `AddOnTemplate` + `AddOnDeploymentConfig`
and supports manual enablement and placement-based auto-install.

## Architecture

- One Deployment per managed cluster (`flock-agent`)
- One container in Pod: `flock-alliance-client`
- Shared volume mounted at `/data` (default hostPath: `/data/flock-client`, optional PVC/emptyDir)
- Client loads env file from mounted volume (default: `/data/.env`)

## Prerequisites

- OCM hub + managed clusters
- A blockchain private key secret on managed clusters
- Per-cluster runtime values provided from mounted `.env` (recommended):
  - `BLOCKCHAIN_RPC`, `TOKEN_ADDRESS`, `TASK_ADDRESS`
- If using `hostPath`, the same absolute path must be available on every schedulable node in each managed cluster

Create secret in each managed cluster install namespace (default: `flock-system`):

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

Override shared addon fields at deploy time:

```bash
helm upgrade --install flock-addon charts/flock-addon \
  --set deploymentConfig.storage.backend='local' \
  --set deploymentConfig.storage.localSharedDir='/data/shared'
```

For multi-cluster deployments, do **not** hardcode
`deploymentConfig.blockchain.tokenAddress/taskAddress` in Helm values when
you want per-cluster values. Keep per-cluster chain settings in each managed
cluster's mounted `.env` file.

For `deploymentConfig.storage.backend=local`, all participants must see the same
shared filesystem path (for example via NFS-backed PVC mounted in each cluster).
Set `agent.dataVolume.existingClaim=<rwx-claim>` (or `hostPath` for single-node dev).

## Use Mounted `.env` (Recommended)

`flock-alliance-client` loads `/data/.env` at startup by default.

- path control: `deploymentConfig.runtime.flockAllianceEnvFile`
- default value: `/data/.env`

Example with hostPath:

```bash
helm upgrade --install flock-addon charts/flock-addon \
  --set agent.dataVolume.hostPath='/data/flock-client' \
  --set deploymentConfig.runtime.flockAllianceEnvFile='/data/.env'
```

Path mapping rules:

- Node path (host filesystem): `HOST_DATA_PATH` (default from chart `agent.dataVolume.hostPath`)
- Container mount path: always `/data`
- Env file path inside container: `FLOCK_ALLIANCE_ENV_FILE` (default `/data/.env`)
- Effective `.env` on node: `${HOST_DATA_PATH}/.env`

`hostPath` should be an absolute node path (for example `/data/flock-client`).
Do not use `~` because kubelet does not expand shell home paths.
Ensure node filesystem permissions allow kubelet and container runtime access.

Prepare this directory on every node that may run the pod:

```bash
sudo mkdir -p /data/flock-client
sudo chmod 755 /data
sudo chown -R ubuntu:ubuntu /data/flock-client
sudo chmod -R u+rwX /data/flock-client
```

If your login user is not `ubuntu`, replace `ubuntu:ubuntu` with your actual
user/group.

Place env file on each managed cluster node:

```text
/data/flock-client/.env
```

Minimal `.env` for direct client:

```dotenv
PRIVATE_KEY=0x...
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
HF_TOKEN=hf_...
```

Priority notes:

- AddOnDeploymentConfig provides defaults.
- If env file exists, startup sourcing can override defaults.
- Non-empty variables become CLI overrides; empty variables are not forced.
- `FLockAlliance` priority remains: CLI overrides > environment > YAML config.

## Per-Cluster Config Override

If one cluster needs different defaults, create a dedicated
`AddOnDeploymentConfig` on hub and reference it from that cluster's
`ManagedClusterAddOn`.

If one cluster uses a different node path, set `HOST_DATA_PATH` in that
cluster's dedicated deployment config.

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

Reference from `ManagedClusterAddOn`:

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
kubectl label managedcluster cluster1 gpu=true
```

or:

```bash
make deploy-auto-all
```

## OCM Validation Checklist

After enabling on a managed cluster:

```bash
kubectl get clustermanagementaddon flock-addon -o yaml
kubectl -n cluster1 get managedclusteraddon flock-addon -o yaml
kubectl -n flock-system get deploy,pod
kubectl -n flock-system logs deploy/flock-agent -c flock-alliance-client --tail=100
```

What to verify:

- `ManagedClusterAddOn` condition shows install/available progressing normally.
- Pod runs `flock-alliance-client` successfully.
- Log shows `runtime.mode=local`.
- If `.env` is mounted, runtime uses expected RPC/address/data-path values.

## Validate Chart

```bash
make verify
make test-chart
```
