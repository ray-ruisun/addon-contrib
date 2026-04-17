# FLock Addon for Decentralized Federated AI

Integrate [FLock FL Alliance](https://github.com/FLock-io/FL-Alliance-Client) (`FLockAlliance`) with Open Cluster Management (OCM) to automate decentralized federated learning across multi-cluster and multi-cloud environments, with blockchain-backed coordination and incentive mechanisms for participating nodes.

## Key Characteristics

- OCM deploys the decentralized `FLockAlliance` client to managed clusters as a direct participant runtime
- One-command, hub-managed deployment flows for testnet and local-chain development, including an optional hub-hosted S3-compatible object store
- Preserves the protocol's incentive-driven workflow, including on-chain task coordination and reward-oriented participation
- Runtime mode is fixed to `local`, so each cluster joins the protocol with its own node-local data and secrets
- `FLocKit` is not deployed as a separate addon workload; training runs inside the same `flock-alliance-client` container process
- Each enabled managed cluster runs one `flock-agent` Deployment in `flock-system`
- Runtime configuration, datasets, and model inputs are loaded from a mounted node directory, usually `/data/flock-client`

## Features

| Capability | Description |
| --- | --- |
| Decentralized FL | Runs `FLockAlliance`, a blockchain-backed federated learning client with on-chain task coordination and incentive-aware participation |
| Deployment | Uses OCM primitives such as `ClusterManagementAddOn`, `AddOnTemplate`, and `AddOnDeploymentConfig` for declarative multi-cluster rollout |
| Runtime Architecture | Keeps the addon simple with one direct `flock-alliance-client` workload per managed cluster and no separate `FLocKit` addon component |
| Placement | Automatically selects a CPU or GPU addon template from the managed cluster label `gpu=true` |
| Data Locality | Reads `.env`, datasets, and model inputs from node-mounted storage so each cluster trains on its own local resources |
| Configuration Authority | Hub-pushed values (task address, storage backend, hub-managed RPC/S3) always win over stale values in node `.env`, regardless of storage backend |
| Modes | Supports testnet, local chain + original S3, and local chain + local S3-compatible storage workflows |

## Supported Deployment Modes

| Mode                              | Command | Best for |
|-----------------------------------| --- | --- |
| Local chain + local S3-compatible | `make deploy-local-chain-s3-compatible` | Recommended default path: fully self-contained deployment with hub-managed local chain and object storage |
| Local chain + original S3         | `make deploy-local-chain-s3` | Use when you want a hub-hosted local chain but still depend on an existing external S3 model artifact |
| Testnet                           | `make deploy-testnet` | Use when you already have an on-chain task and shared external S3 workflow ready on testnet |

Recommended default: start with `Local chain + local S3-compatible`. It keeps the blockchain and storage dependencies inside the managed environment, while the other two modes depend on an existing on-chain task and/or external S3 storage.

Full mode details are in [Deployment Modes](docs/deployment-modes.md).

## Architecture

```text
┌─────────────────────────────────────────────────────────────┐
│                        Hub Cluster                          │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  ClusterManagementAddOn + AddOnTemplate (CPU / GPU)    │ │
│  │  AddOnDeploymentConfig  (CPU / GPU)                    │ │
│  │  [optional] local anvil chain + MinIO (self-contained) │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Managed Cluster                         │
│  namespace: flock-system                                    │
│  Deployment: flock-agent                                    │
│  Container:  flock-alliance-client                          │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  Managed Cluster Node                       │
│  hostPath: /data/flock-client                               │
│  files:    .env, datasets, model inputs                     │
└─────────────────────────────────────────────────────────────┘
```

## Documentation

- [Prepare Multi-Cluster Environment](docs/prepare-multicluster-environment.md) - build Kubernetes clusters, install OCM, register managed clusters, and verify ManifestWork distribution
- [Install FLock Addon](docs/install-flock-addon.md) - first deployment path for the recommended default workflow
- [Deployment Modes](docs/deployment-modes.md) - compare and run the three supported deployment modes
- [Image Management](docs/image-management.md) - choose public/private images and publish custom builds
- [Configuration and Overrides](docs/configuration-and-overrides.md) - runtime model, path rules, task updates, and per-cluster overrides
- [Troubleshooting](docs/troubleshooting.md) - image pull, OCM distribution, GPU mapping, and log collection
- [Tests](tests/README.md) - chart unit tests and entrypoint shell tests that gate every change

## Related Projects

- [FLock FL Alliance Client](https://github.com/FLock-io/FL-Alliance-Client) - the direct client runtime deployed by this addon
- [FLock FLocKit](https://github.com/FLock-io/FLocKit) - the federated learning training logic template
- [Open Cluster Management](https://open-cluster-management.io) - multi-cluster management for Kubernetes
