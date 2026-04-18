# Deployment Modes

`flock-addon` supports three deployment modes. The recommended default is `deploy-local-chain-s3-compatible`, because it keeps both blockchain and storage under hub-managed control and avoids depending on an existing on-chain task or external S3 bucket.

## Mode Summary

| Mode | Command | Blockchain source | Storage backend | Model input |
| --- | --- | --- | --- | --- |
| Local chain + local S3-compatible | `make deploy-local-chain-s3-compatible` | Hub starts local chain | `nami` | Local `MODEL_ARCHIVE` uploaded by the hub |
| Local chain + original S3 | `make deploy-local-chain-s3` | Hub starts local chain | `s3` | Existing uploaded `MODEL_HASH` |
| Testnet | `make deploy-testnet` | Existing external RPC from node `.env` | `s3` | Existing onchain task |

## Recommendation

Start with `deploy-local-chain-s3-compatible` unless you already need a shared external environment.

Why this is the default recommendation:

- it does not require you to create a task on a public testnet first
- it does not depend on a shared external S3 bucket
- the hub manages the local chain and local S3-compatible object storage for you
- it is the most self-contained path for cluster-level validation and iterative development

Choose the other modes only when their external dependencies are already part of your workflow:

- `deploy-testnet` requires an existing on-chain task and the testnet-oriented external storage flow
- `deploy-local-chain-s3` still depends on an existing external S3 model artifact even though the chain is local

## Environment Requirements by Mode

| Variable | `deploy-testnet` | `deploy-local-chain-s3` | `deploy-local-chain-s3-compatible` |
| --- | --- | --- | --- |
| `PRIVATE_KEY` | required | required | required |
| `HF_TOKEN` | required | required | required |
| `BLOCKCHAIN_RPC` | required from node `.env` | pushed from the hub | pushed from the hub |
| `TOKEN_ADDRESS` | optional from node `.env` when the hub leaves it empty | pushed from the hub | pushed from the hub |
| `S3_COMPAT_ENDPOINT_URL` | not used | not used | pushed from the hub |
| `S3_COMPAT_BUCKET` | not used | not used | pushed from the hub |
| `S3_COMPAT_ACCESS_KEY` | not used | not used | pushed from the hub |
| `S3_COMPAT_SECRET_KEY` | not used | not used | pushed from the hub |
| `S3_COMPAT_REGION` | not used | not used | pushed from the hub |
| `S3_COMPAT_ADDRESSING_STYLE` | not used | not used | pushed from the hub |
| `S3_COMPAT_VERIFY_SSL` | not used | not used | pushed from the hub |

## Testnet Mode

Use this mode only when you already have a ready testnet task and the external storage workflow in place.

```bash
# [Hub]
make deploy-testnet TASK_ADDRESS='0x<task-address>'
```

Node `.env` example:

```dotenv
PRIVATE_KEY=<private-key>
HF_TOKEN=<hf-token>
BLOCKCHAIN_RPC=<testnet-rpc-url>
TOKEN_ADDRESS=0x<token-address>
```

This flow is covered step by step in [Install FLock Addon](install-flock-addon.md).

Common rollout pattern after the hub-side deploy:

```bash
# [Hub]
make enable-addon CLUSTER=<cluster-a>
make enable-addon CLUSTER=<cluster-b>
make enable-addon CLUSTER=<cluster-c>
```

## Local Chain + Original S3 Mode

Use this mode when:

- the hub should auto-start the local chain
- storage should stay on the original signer-based `s3` backend
- the hub should push the local-chain `BLOCKCHAIN_RPC`
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
  RPC_HOST=<hub-ip> \
  DOCKER='sudo docker'
```

Important:

- wait for the command to finish by itself
- do not press `Ctrl+C` while `make chain` is still deploying contracts
- `anvil` is started in the background, but the hub still waits for the one-shot `deployer` step to finish
- if you interrupt this step early, `data/contracts.json` may be missing or incomplete, and `TOKEN_ADDRESS` or `TASK_ADDRESS` will not be pushed to the addon

Managed cluster node `.env` for this mode only needs node-local secrets:

```dotenv
PRIVATE_KEY=<private-key>
HF_TOKEN=<hf-token>
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
- `TASK_ADDRESS` matches the hub-generated value
- the deployed task was created from the `MODEL_HASH` you passed to `make chain`

After the hub-side deploy finishes:

- Clusters that are **already enabled** pick up the new `BLOCKCHAIN_RPC`, `TASK_ADDRESS`, and `TOKEN_ADDRESS` automatically. The OCM addon-manager detects the bumped `AddOnDeploymentConfig.metadata.generation`, re-renders the per-cluster `ManifestWork`, and the FL client Pod rolls with the new env. No operator action is required; verify with:

  ```bash
  # [Hub]
  kubectl -n <cluster> get managedclusteraddon flock-addon -o yaml | grep -A2 lastObservedGeneration
  kubectl --context=<managed> -n flock-system get pod -l app.kubernetes.io/name=flock-addon
  ```

- Clusters that have **never been enabled** still need a one-time opt-in:

  ```bash
  # [Hub]
  make enable-addon CLUSTER=<cluster-a>
  ```

- Force an immediate Pod restart only if you suspect reconcile is stuck or you are switching CPU↔GPU templates (see [Troubleshooting](troubleshooting.md)):

  ```bash
  # [Hub] — equivalent to deleting + recreating the ManagedClusterAddOn
  make disable-addon CLUSTER=<cluster-a>
  make enable-addon CLUSTER=<cluster-a>
  ```

## Local Chain + Local S3-Compatible Storage Mode

Use this mode when the hub should host both:

- the local chain
- a local S3-compatible object store such as MinIO

Important:

- this mode uses `storage.backend=nami`
- the hub starts local S3-compatible storage automatically
- the hub uploads `MODEL_ARCHIVE` automatically
- the hub pushes the local-chain `BLOCKCHAIN_RPC`
- the hub auto-pushes `S3_COMPAT_ENDPOINT_URL`, `S3_COMPAT_BUCKET`, `S3_COMPAT_ACCESS_KEY`, `S3_COMPAT_SECRET_KEY`, `S3_COMPAT_REGION`, `S3_COMPAT_ADDRESSING_STYLE`, and `S3_COMPAT_VERIFY_SSL`
- nodes only need their own local secrets such as `PRIVATE_KEY` and `HF_TOKEN`

### Ephemeral, task-scoped local storage

Each `make deploy-local-chain-s3-compatible` invocation provisions a brand-new task. To keep operators safe and tasks isolated, the hub generates the following per task and propagates them through the AddOnDeploymentConfig so every managed cluster receives the *same* values:

- a fresh access/secret pair (16/24-byte hex from `openssl rand`) — overrides the legacy `minioadmin` defaults
- a task-scoped bucket name `flock-task-<sha256[:12]>` derived from `MODEL_ARCHIVE`

Multi-client coordination is preserved exactly: every FL client Pod still resolves the same `S3_COMPAT_BUCKET`, `S3_COMPAT_ACCESS_KEY`, and `S3_COMPAT_SECRET_KEY` because OCM substitutes those customizedVariables identically on every cluster. The only difference is that the value being substituted is now task-unique instead of a well-known constant.

If you need to pin known values (e.g. for repeatable CI smoke tests), pass them on the make command line and the hub will skip the random generation:

```bash
# [Hub] (optional; defaults are auto-generated)
make deploy-local-chain-s3-compatible \
  ... \
  MINIO_ACCESS_KEY=ci-access \
  MINIO_SECRET_KEY=ci-secret \
  MINIO_BUCKET=flock-ci
```

Hub reboots while a local-chain task is running mean the chain process and MinIO container are both lost; re-create the task from scratch with `make deploy-local-chain-s3-compatible`.

Managed cluster node `.env`:

```dotenv
PRIVATE_KEY=<private-key>
HF_TOKEN=<hf-token>
```

Prepare the archive you want the hub to upload, for example:

```bash
git clone https://github.com/FLock-io/FLocKit.git
cd FLocKit
tar -czf ../model.tar.gz .
```

Deploy:

```bash
# [Hub]
make deploy-local-chain-s3-compatible \
  FL_ALLIANCE_CLIENT_DIR=/path/to/FL-Alliance-Client \
  MODEL_ARCHIVE=/path/to/model.tar.gz \
  RPC_HOST=<hub-ip> \
  DOCKER='sudo docker' \
  S3_COMPAT_DATA_DIR='<local-minio-data-dir>'
```

`S3_COMPAT_DATA_DIR` defaults to `/srv/flock-minio/data`. Override it if your normal user cannot write there. If you keep the default path, create it first:

```bash
sudo mkdir -p /srv/flock-minio/data
sudo chown -R "$USER":"$(id -gn)" /srv/flock-minio
```

Important:

- wait for the command to finish by itself
- do not press `Ctrl+C` while `make chain` is still deploying contracts
- this mode also waits for the local MinIO upload and the one-shot `deployer` step before Helm deploy starts
- if you interrupt this step early, `data/contracts.json` may be missing or incomplete, and the addon will not get valid `TOKEN_ADDRESS` or `TASK_ADDRESS`

Optional manual local S3-compatible service on the hub (only useful for ad-hoc `mc cp` debugging — `make deploy-local-chain-s3-compatible` already starts and configures MinIO with task-scoped credentials):

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

To recover the credentials and bucket name of the currently deployed task:

```bash
# [Hub]
make minio-credentials
# Prints ENDPOINT_URL=, BUCKET=, ACCESS_KEY=, SECRET_KEY= and an `mc alias set` snippet.
```

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
- `TASK_ADDRESS` matches the hub-generated value
- `S3_COMPAT_ENDPOINT_URL` points to `http://<hub-ip>:9000`
- client logs include `Using direct S3-compatible storage backend`

After the hub-side deploy finishes, the same rule as the previous mode applies — already-enabled clusters reconcile onto the new bucket, credentials, RPC, and task addresses without operator action; new clusters need a one-time `make enable-addon CLUSTER=<name>`. Use `make disable-addon` + `make enable-addon` only to force an immediate Pod restart or to switch CPU↔GPU templates:

```bash
# [Hub]
make enable-addon CLUSTER=<cluster-a>   # one-time, only if never enabled
make enable-addon CLUSTER=<cluster-b>
make enable-addon CLUSTER=<cluster-c>
```

## How Runtime Values Are Chosen

The entrypoint uses a single, backend-independent rule: whenever the hub pushes a non-empty value for a variable, that hub value wins over anything the node `.env` might set. Variables the hub leaves empty fall through to the node `.env` and, if still unset, to the FLockAlliance YAML defaults. Full details are in [Configuration and Overrides](configuration-and-overrides.md).

Practical consequence per mode:

- Testnet: the hub keeps `BLOCKCHAIN_RPC` empty, so each node `.env` supplies it. `TASK_ADDRESS`, `STORAGE_BACKEND=s3`, and the GPU selection stay hub-authoritative.
- `deploy-local-chain-s3`: the hub pushes `BLOCKCHAIN_RPC`, `TOKEN_ADDRESS`, and `TASK_ADDRESS`; `STORAGE_BACKEND=s3`.
- `deploy-local-chain-s3-compatible`: same as above plus every `S3_COMPAT_*` setting, and `STORAGE_BACKEND=nami`.

## Cleanup

```bash
# [Hub]
# Remove only the Helm release (managed cluster workloads are garbage-collected by OCM):
make undeploy

# Remove the Helm release AND stop the local MinIO container started by
# deploy-local-chain-s3-compatible (the data directory is left on disk):
make undeploy-all

# Tear down a complete local-chain task: drop the per-task MinIO bucket,
# uninstall the Helm release, stop the MinIO container, and (when
# FL_ALLIANCE_CLIENT_DIR is provided) attempt to stop the local chain.
# Use this between successive `deploy-local-chain-*` invocations so you
# never leave stale per-task buckets on disk.
make undeploy-local-chain FL_ALLIANCE_CLIENT_DIR=/path/to/FL-Alliance-Client
```
