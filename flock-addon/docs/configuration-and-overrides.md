# Configuration and Overrides

This guide explains where `flock-addon` runs, how runtime values are resolved, and how to override defaults for specific clusters.

## What Runs Where

### Hub cluster

- Stores `ClusterManagementAddOn`, `AddOnTemplate`, and `AddOnDeploymentConfig`
- Deploys and updates shared addon settings
- Enables the addon on selected managed clusters (Manual) or auto-installs via `Placement` (Placements)

### Managed cluster

- Receives `ManifestWork` from the hub addon manager
- Runs the `flock-agent` Deployment in namespace `flock-system`

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

- do not use `~` in `hostPath`; always use an absolute path such as `/data/flock-client`
- the same host path must exist on every node that may schedule the Pod
- if your GPU nodes use taints or dedicated labels, set `agent.tolerations` and/or `agent.nodeSelector` via Helm

## Update Task Address

When a new onchain task is created, update only `TASK_ADDRESS`:

```bash
# [Hub]
make update-task TASK_ADDRESS='0x<NEW_TASK_ADDRESS>'
```

Check:

```bash
# [Hub]
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config -o yaml | rg -n "TASK_ADDRESS|value"
```

If the addon is already enabled, the managed cluster workload should reconcile automatically because the AddOnDeploymentConfig has changed. If you want to force a refresh immediately:

```bash
# [Hub]
make disable-addon CLUSTER=<cluster-name>
make enable-addon CLUSTER=<cluster-name>
```

## Parameter Flow

1. Helm values render shared `AddOnDeploymentConfig` objects (CPU + GPU variants).
2. `ManagedClusterAddOn` selects one `AddOnTemplate` and one `AddOnDeploymentConfig` per cluster.
3. OCM injects `customizedVariables` into the template placeholders (`{{FLOCK_ALLIANCE_IMAGE}}`, `{{TASK_ADDRESS}}`, ...).
4. The Pod receives those values as environment variables (`TASK_ADDRESS`, `USE_GPU`, `HOST_DATA_PATH`, ...).
5. The container entrypoint:
   - snapshots every hub-pushed value into an `OCM_*` shadow variable
   - sources `FLOCK_ALLIANCE_ENV_FILE` (default `/data/.env`) so per-node secrets become available
   - **re-exports every `OCM_*` value that is non-empty** so hub-pushed values always win, regardless of storage backend
   - validates `STORAGE_BACKEND` is one of `s3|local|nami` and warns when `TASK_ADDRESS` is empty
   - builds and `exec`'s `python -u main.py --config config/conf.yaml --override ...`

Effective authority rules:

- hub value non-empty → hub wins (for `TASK_ADDRESS`, `USE_GPU`, `STORAGE_BACKEND`, `NO_INCENTIVE`, `NUM_PARTICIPANTS`, `STAKE`, `BLOCKCHAIN_RPC`, `TOKEN_ADDRESS`, `LOCAL_STORAGE_DIR`, and all `S3_COMPAT_*`)
- hub value empty → the value from the node `.env` (if any) is used
- neither set → the FLockAlliance YAML default applies

This means:

- in testnet mode, the hub leaves `BLOCKCHAIN_RPC` empty so each node `.env` provides it; the hub stays authoritative for `TASK_ADDRESS`, `STORAGE_BACKEND`, and GPU selection
- in hub-managed local-chain modes, the hub pushes `BLOCKCHAIN_RPC`, `TOKEN_ADDRESS`, and - when running `nami` - every `S3_COMPAT_*` setting, and those values always win

## Per-Cluster Override

If one cluster needs different defaults, create a dedicated `AddOnDeploymentConfig` and reference it from that cluster's `ManagedClusterAddOn`.

Example `AddOnDeploymentConfig`:

```yaml
apiVersion: addon.open-cluster-management.io/v1alpha1
kind: AddOnDeploymentConfig
metadata:
  name: flock-addon-config-<cluster-name>
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
  namespace: <cluster-name>
  annotations:
    addon.open-cluster-management.io/v1alpha1-install-namespace: flock-system
spec:
  configs:
    - group: addon.open-cluster-management.io
      resource: addontemplates
      name: flock-addon
    - group: addon.open-cluster-management.io
      resource: addondeploymentconfigs
      name: flock-addon-config-<cluster-name>
      namespace: open-cluster-management
```

Because the addon entrypoint treats any non-empty hub value as authoritative, leaving a field empty in a per-cluster `AddOnDeploymentConfig` lets the node `.env` provide it. Set the field to an explicit value when you want the hub to win for that cluster.

## Direct CLI Mapping

Old direct run:

```bash
python main.py \
  --task-address 0x<task-address> \
  --dataset /path/to/dataset \
  --hf-token <hf-token> \
  --gpu
```

Addon mapping:

- `--task-address` → `deploymentConfig.blockchain.taskAddress` (hub) or `TASK_ADDRESS` in node `.env`
- `--dataset`      → `DATA_PATH` (hub, pointing at the in-container mount of `agent.dataVolume.*`)
- `--hf-token`     → `HF_TOKEN` in node `.env`
- `--gpu`          → GPU `AddOnTemplate` + `AddOnDeploymentConfig` selected automatically by `make enable-addon` when the managed cluster has label `gpu=true`

Effective priority inside the client, highest to lowest:

1. CLI `--override` flags built by the addon entrypoint
2. Environment variables (hub `customizedVariables` with the "hub-wins-when-non-empty" rule described above, plus node `.env`)
3. `config/conf.yaml` defaults shipped with FLockAlliance
