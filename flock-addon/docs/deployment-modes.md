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
make enable-addon CLUSTER=m1
make enable-addon CLUSTER=m2
make enable-addon CLUSTER=m3
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

After the hub-side deploy finishes, re-enable the addon on each target managed cluster so it reconciles onto the new config:

```bash
# [Hub]
make disable-addon CLUSTER=m1
make enable-addon CLUSTER=m1
make disable-addon CLUSTER=m2
make enable-addon CLUSTER=m2
make disable-addon CLUSTER=m3
make enable-addon CLUSTER=m3
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

Optional manual local S3-compatible service on the hub:

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

After the hub-side deploy finishes, re-enable the addon on each target managed cluster:

```bash
# [Hub]
make disable-addon CLUSTER=m1
make enable-addon CLUSTER=m1
make disable-addon CLUSTER=m2
make enable-addon CLUSTER=m2
make disable-addon CLUSTER=m3
make enable-addon CLUSTER=m3
```

## How Runtime Values Are Chosen

- In testnet mode, `BLOCKCHAIN_RPC` comes from each node `.env`
- In `deploy-local-chain-s3`, the hub pushes `BLOCKCHAIN_RPC`, `TOKEN_ADDRESS`, and `TASK_ADDRESS`
- In `deploy-local-chain-s3-compatible`, the hub also pushes S3-compatible settings
- `TASK_ADDRESS`, `USE_GPU`, `STORAGE_BACKEND`, and `NO_INCENTIVE` stay authoritative from OCM in all modes
