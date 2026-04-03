# Configuration and Overrides

This guide explains where `flock-addon` runs, how runtime values are resolved, and how to override defaults for specific clusters.

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
- Stores `.env` and any local datasets or files used by `FLockAlliance`

## Runtime Model

Each managed cluster gets one Pod with one container:

- Deployment: `flock-agent`
- Container: `flock-alliance-client`
- Container mount path: `/data`
- Default node path: `/data/flock-client`
- Default env file inside container: `/data/.env`
- Effective env file on node: `/data/flock-client/.env`

Path rules:

- do not use `~` in `hostPath`
- use an absolute path such as `/data/flock-client`
- the same host path must exist on every node that may schedule the Pod
- if your GPU nodes use taints or dedicated labels, set `agent.tolerations` and/or `agent.nodeSelector`

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

If the addon is already enabled, the managed cluster workload should reconcile automatically. If you want to force a refresh:

```bash
# [Hub]
make disable-addon CLUSTER=cluster1
make enable-addon CLUSTER=cluster1
```

## Parameter Flow

1. Helm values render shared `AddOnDeploymentConfig` objects.
2. `ManagedClusterAddOn` selects one `AddOnTemplate` and one `AddOnDeploymentConfig`.
3. OCM injects `customizedVariables` into the template placeholders.
4. The Pod gets env vars such as `TASK_ADDRESS`, `USE_GPU`, and `HOST_DATA_PATH`.
5. The container entrypoint loads `.env` from `FLOCK_ALLIANCE_ENV_FILE`.
6. The entrypoint appends CLI `--override` values before starting `FLockAlliance`.

In practice:

- `TASK_ADDRESS`, `USE_GPU`, `STORAGE_BACKEND`, and `NO_INCENTIVE` stay authoritative from OCM
- in testnet mode, `BLOCKCHAIN_RPC` comes from each node `.env`
- in hub-managed chain modes, `BLOCKCHAIN_RPC`, `TOKEN_ADDRESS`, and the selected storage settings are pushed from OCM

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
- `--gpu` maps to selecting the GPU addon template and config for `gpu=true` clusters

Effective priority:

- CLI overrides
- environment variables
- YAML config defaults
