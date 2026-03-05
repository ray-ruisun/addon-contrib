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

This allows frequent model/template changes without modifying FLockAlliance code.

## Validate Chart

```bash
make verify
make test-chart
```
