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
- Default testnet behavior:
  - `deploymentConfig.blockchain.taskAddress` is required at deploy/upgrade time
  - Other chain values (for example `BLOCKCHAIN_RPC`, `TOKEN_ADDRESS`) are read from each cluster's mounted `.env`
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

For testnet onchain mode, use `make deploy-testnet` (it requires `TASK_ADDRESS`).

Override shared addon fields at deploy time:

```bash
helm upgrade --install flock-addon charts/flock-addon \
  --set deploymentConfig.storage.backend='local' \
  --set deploymentConfig.storage.localSharedDir='/data/shared'
```

For multi-cluster deployments:

- Testnet default pattern:
  - Set only `deploymentConfig.blockchain.taskAddress` from Helm values
  - Keep `BLOCKCHAIN_RPC` and `TOKEN_ADDRESS` in each cluster's mounted `.env`
- If you want fully centralized chain settings, you can also set
  `deploymentConfig.blockchain.rpc/tokenAddress` in Helm values.

For `deploymentConfig.storage.backend=local`, all participants must see the same
shared filesystem path (for example via NFS-backed PVC mounted in each cluster).
Set `agent.dataVolume.existingClaim=<rwx-claim>` (or `hostPath` for single-node dev).

## Chain Settings

### 1) Common Settings (Both Modes)

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

Common `.env` template:

```dotenv
HF_TOKEN=hf_...
# Optional per-node overrides:
# For default testnet flow, set RPC + TOKEN here.
# BLOCKCHAIN_RPC=
# TOKEN_ADDRESS=
# STORAGE_BACKEND=s3
# LOCAL_STORAGE_DIR=/data/shared
```

Notes:

- `PRIVATE_KEY` is injected from Kubernetes Secret (`flock-alliance-secret`), not required in `.env`.
- In testnet mode, `TASK_ADDRESS` should be passed from deployment settings per run.

### 2) Testnet Onchain Mode

Use a public testnet RPC and testnet contract addresses.

Default rule in this addon:

- `taskAddress` is required from deployment settings
- `BLOCKCHAIN_RPC` and `TOKEN_ADDRESS` are read from each cluster's mounted `.env`

Testnet deploy command:

```bash
helm upgrade --install flock-addon charts/flock-addon \
  --set agent.dataVolume.hostPath='/data/flock-client' \
  --set deploymentConfig.runtime.flockAllianceEnvFile='/data/.env' \
  --set deploymentConfig.blockchain.taskAddress='0x47B0397C6ae306002788D093b29bcD2EDAd19924' \
  --set deploymentConfig.storage.backend='s3'
```

Equivalent Make target:

```bash
make deploy-testnet \
  TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
```

Optional: if you want to override `.env` values from hub-side settings, pass
`RPC=...` and/or `TOKEN_ADDRESS=...` to `make deploy-testnet`.

`deploymentConfig.blockchain.taskAddress` is passed at startup as a runtime
override (equivalent to direct client `--task-address ...`).
When `storage.backend=s3`, client uses S3 signer mode by default.

When a new task is created, update only `taskAddress`:

```bash
helm upgrade flock-addon charts/flock-addon \
  --reuse-values \
  --set deploymentConfig.blockchain.taskAddress='0x<NEW_TASK_ADDRESS>'
```

Equivalent Make target:

```bash
make update-task TASK_ADDRESS='0x<NEW_TASK_ADDRESS>'
```

Example testnet `.env`:

```dotenv
HF_TOKEN=hf_...
BLOCKCHAIN_RPC=https://sepolia.base.org
TOKEN_ADDRESS=0x...
# TASK_ADDRESS is passed by deploymentConfig.blockchain.taskAddress
# STORAGE_BACKEND defaults to s3 in deploy-testnet
```

### 3) Local Chain Mode

Use a local chain RPC and contract addresses from your local deployment.

Important: do not use `127.0.0.1` unless chain runs in the same Pod.
In most cases, use a node IP or in-cluster Service DNS reachable by the addon Pod.

Example `.env`:

```dotenv
BLOCKCHAIN_RPC=http://<node-ip-or-service>:8545
TOKEN_ADDRESS=0x...
TASK_ADDRESS=0x...
STORAGE_BACKEND=local
LOCAL_STORAGE_DIR=/data/shared
HF_TOKEN=hf_...
```

Deploy command (local storage example):

```bash
helm upgrade --install flock-addon charts/flock-addon \
  --set agent.dataVolume.hostPath='/data/flock-client' \
  --set deploymentConfig.runtime.flockAllianceEnvFile='/data/.env' \
  --set deploymentConfig.storage.backend='local' \
  --set deploymentConfig.storage.localSharedDir='/data/shared'
```

### CLI Mapping (Old Command -> Addon)

Old direct command example:

```bash
python main.py \
  --task-address 0x47B0397C6ae306002788D093b29bcD2EDAd19924 \
  --dataset data/asr_sarawakmalay_whisper_format_client_ids.json \
  --hf-token $HF_TOKEN \
  --gpu
```

Mapping in addon deployment:

- `--task-address ...` -> `deploymentConfig.blockchain.taskAddress` (hub-side shared setting)
- `--dataset ...` -> `DATA_PATH` (set `agent.dataPath`, can be file path or directory, default `/data`)
- `--hf-token ...` -> `HF_TOKEN` secret key (`flock-alliance-secret/HF_TOKEN`) or `.env`
- `--gpu` -> `deploymentConfig.runtime.useGpu=true`

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
