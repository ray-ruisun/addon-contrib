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
- Default testnet behavior:
  - `deploymentConfig.blockchain.taskAddress` is required at deploy/upgrade time
  - Other chain values (for example `BLOCKCHAIN_RPC`, `TOKEN_ADDRESS`) are read from each cluster's mounted `.env`
- If using `hostPath`, the same absolute path must be available on every schedulable node in each managed cluster

## Deploy

Run on: **Hub cluster**.

```bash
cd flock-addon
make deploy
```

Check:

```bash
# [Hub]
kubectl get clustermanagementaddon flock-addon
kubectl -n open-cluster-management get addontemplate flock-addon
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config
```

Should see:
- `clustermanagementaddon/flock-addon` exists
- `addontemplate/flock-addon` exists
- `addondeploymentconfig/flock-addon-config` exists

Default image used by this addon:

- `ghcr.io/ray-ruisun/fl-alliance-client:v0.1.0`
- if this package is private, also configure `image.pullSecrets`

To temporarily override the image for one deployment, use an environment variable:

```bash
# [Hub]
FLOCK_ALLIANCE_IMAGE='ghcr.io/<owner>/fl-alliance-client:<tag>' make deploy
```

Or for testnet:

```bash
# [Hub]
FLOCK_ALLIANCE_IMAGE='ghcr.io/<owner>/fl-alliance-client:<tag>' \
make deploy-testnet TASK_ADDRESS='0x...'
```

You can also override directly in Helm:

```bash
# [Hub]
helm upgrade --install flock-addon charts/flock-addon \
  --set image.repository='ghcr.io/<owner>/fl-alliance-client' \
  --set image.tag='<tag>'
```

If the image registry requires authentication, also pass pull secrets:

```bash
# [Hub]
helm upgrade --install flock-addon charts/flock-addon \
  --set image.repository='ghcr.io/<owner>/fl-alliance-client' \
  --set image.tag='<tag>' \
  --set image.pullSecrets[0]='ghcr-creds'
```

For testnet onchain mode, use `make deploy-testnet` (it requires `TASK_ADDRESS`).

Override shared addon fields at deploy time:

```bash
# [Hub]
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
# [Hub]
helm upgrade --install flock-addon charts/flock-addon \
  --set agent.dataVolume.hostPath='/data/flock-client'
```

Path mapping rules:

- Node path (host filesystem): `HOST_DATA_PATH` (default from chart `agent.dataVolume.hostPath`)
- Container mount path: always `/data`
- Env file path inside container: `FLOCK_ALLIANCE_ENV_FILE` (default `/data/.env`)
- Effective `.env` on node: `${HOST_DATA_PATH}/.env`
- Recommended node `.env` path: `/data/flock-client/.env`

Important:
- Keep `FLOCK_ALLIANCE_ENV_FILE` as `/data/.env` (container path).
- Put the actual file on node at `/data/flock-client/.env` (host path).

`hostPath` should be an absolute node path (for example `/data/flock-client`).
Do not use `~` because kubelet does not expand shell home paths.
Ensure node filesystem permissions allow kubelet and container runtime access.

Prepare this directory on every node that may run the pod:

```bash
# [Each Managed Cluster Node]
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
PRIVATE_KEY=0x...
HF_TOKEN=hf_...
# Optional per-node overrides:
# For default testnet flow, set RPC + TOKEN here.
# BLOCKCHAIN_RPC=
# TOKEN_ADDRESS=
# STORAGE_BACKEND=s3
# LOCAL_STORAGE_DIR=/data/shared
```

Notes:

- `PRIVATE_KEY` and `HF_TOKEN` are loaded from mounted `.env`.
- In testnet mode, `TASK_ADDRESS` should be passed from deployment settings per run.

### 2) Testnet Onchain Mode

Use a public testnet RPC and testnet contract addresses.

Default rule in this addon:

- `taskAddress` is required from deployment settings
- `BLOCKCHAIN_RPC` and `TOKEN_ADDRESS` are read from each cluster's mounted `.env`
- GPU is enabled by default (`deploymentConfig.runtime.useGpu=true`)

Testnet deploy command:

```bash
# [Hub]
helm upgrade --install flock-addon charts/flock-addon \
  --set agent.dataVolume.hostPath='/data/flock-client' \
  --set deploymentConfig.blockchain.taskAddress='0x47B0397C6ae306002788D093b29bcD2EDAd19924' \
  --set deploymentConfig.storage.backend='s3'
```

Check:

```bash
# [Hub]
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config -o yaml | rg -n "TASK_ADDRESS|STORAGE_BACKEND|value"
```

Should see:
- `TASK_ADDRESS` value is the one you set
- `STORAGE_BACKEND` is `s3`

Equivalent Make target:

```bash
# [Hub]
make deploy-testnet \
  TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
```

Check:

```bash
# [Hub]
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config -o yaml | rg -n "TASK_ADDRESS|value"
```

Should see:
- deployment config includes the new `TASK_ADDRESS`

To verify it is **really running** (not only configured on Hub):

```bash
# [Hub] confirm addon is enabled to the managed cluster
kubectl -n <cluster-name> get managedclusteraddon flock-addon
kubectl -n <cluster-name> get manifestwork
```

```bash
# [Managed Cluster context] confirm runtime workload
kubectl -n flock-system get deploy,pod
kubectl -n flock-system logs deploy/flock-agent -c flock-alliance-client --tail=100
```

Should see:
- Hub has `managedclusteraddon/flock-addon` and related ManifestWork
- Managed cluster has `deploy/flock-agent`
- Pod is `Running` and logs show client startup

If Pod status is `ImagePullBackOff` or `ErrImagePull`, check the image reference first:

```bash
# [Managed Cluster context]
kubectl -n flock-system describe pod -l app.kubernetes.io/name=flock-addon
```

```bash
# [Hub]
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config -o yaml | rg -n "FLOCK_ALLIANCE_IMAGE|value"
```

Should see:
- Pod events show the exact image pull error
- `FLOCK_ALLIANCE_IMAGE` matches the repository and tag you expect

If the event says `unauthorized` or `denied`, the registry needs credentials.
In that case:

```bash
# [Managed Cluster context]
kubectl -n flock-system create secret docker-registry ghcr-creds \
  --docker-server=ghcr.io \
  --docker-username='<github-user>' \
  --docker-password='<github-token>' \
  --docker-email='<email>'
```

```bash
# [Hub]
helm upgrade --install flock-addon charts/flock-addon \
  --set image.repository='ghcr.io/ray-ruisun/fl-alliance-client' \
  --set image.tag='v0.1.0' \
  --set image.pullSecrets[0]='ghcr-creds'
```

Check:

```bash
# [Managed Cluster context]
kubectl -n flock-system get secret ghcr-creds
kubectl -n flock-system get deploy flock-agent -o yaml | rg -n "imagePullSecrets|ghcr-creds"
```

Should see:
- secret `ghcr-creds` exists
- `deployment/flock-agent` references `imagePullSecrets: ghcr-creds`

If the default image is not accessible, redeploy with an explicit override:

```bash
# [Hub]
FLOCK_ALLIANCE_IMAGE='ghcr.io/ray-ruisun/fl-alliance-client:v0.1.0' \
make deploy-testnet TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
```

Then re-enable the addon to refresh the managed cluster workload:

```bash
# [Hub]
make disable-addon CLUSTER=<cluster-name>
make enable-addon CLUSTER=<cluster-name>
```

Check:

```bash
# [Managed Cluster context]
kubectl -n flock-system get deploy,pod
```

Should see:
- `deployment/flock-agent` becomes `1/1`
- Pod status becomes `Running`

Optional: if you want to override `.env` values from hub-side settings, pass
`RPC=...` and/or `TOKEN_ADDRESS=...` to `make deploy-testnet`.
If you need CPU-only mode, set `--set deploymentConfig.runtime.useGpu='false'`.

`deploymentConfig.blockchain.taskAddress` is passed at startup as a runtime
override (equivalent to direct client `--task-address ...`).
When `storage.backend=s3`, client uses S3 signer mode by default.

When a new task is created, update only `taskAddress`:

```bash
# [Hub]
helm upgrade flock-addon charts/flock-addon \
  --reuse-values \
  --set deploymentConfig.blockchain.taskAddress='0x<NEW_TASK_ADDRESS>'
```

Check:

```bash
# [Hub]
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config -o yaml | rg -n "TASK_ADDRESS|value"
```

Should see:
- `TASK_ADDRESS` is updated to `0x<NEW_TASK_ADDRESS>`

Equivalent Make target:

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
- `TASK_ADDRESS` in deployment config matches your new value

Example testnet `.env`:

```dotenv
PRIVATE_KEY=0x...
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
PRIVATE_KEY=0x...
BLOCKCHAIN_RPC=http://<node-ip-or-service>:8545
TOKEN_ADDRESS=0x...
TASK_ADDRESS=0x...
STORAGE_BACKEND=local
LOCAL_STORAGE_DIR=/data/shared
HF_TOKEN=hf_...
```

Deploy command (local storage example):

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
- `--hf-token ...` -> `HF_TOKEN` in mounted `.env`
- `--gpu` -> default is already `deploymentConfig.runtime.useGpu=true`

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

## Enable Addon On A Cluster

Run on: **Hub cluster**.

```bash
make enable-addon CLUSTER=cluster1
```

Check:

```bash
# [Hub]
kubectl -n cluster1 get managedclusteraddon flock-addon
kubectl -n cluster1 get managedclusteraddon flock-addon -o yaml
kubectl -n cluster1 get manifestwork
```

Should see:
- `managedclusteraddon/flock-addon` exists in `cluster1` namespace
- `spec.configs` includes `flock-addon` and `flock-addon-config`
- status conditions progress toward healthy/available
- one or more ManifestWork objects exist

Then confirm agent runtime on managed cluster:

```bash
# [Managed Cluster context]
kubectl -n flock-system get deploy,pod
kubectl -n flock-system logs deploy/flock-agent -c flock-alliance-client --tail=100
```

Should see:
- `deploy/flock-agent` exists and pod is `Running`
- logs show `flock-alliance-client` startup

## Placement Auto-Install

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
- placement exists and targets `gpu=true` clusters
- labeled cluster eventually gets `managedclusteraddon/flock-addon`

or:

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
- placement exists and targets the configured clusterset
- selected clusters eventually get `managedclusteraddon/flock-addon`

## Validate Chart

```bash
make verify
make test-chart
```
