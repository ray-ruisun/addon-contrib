# FLock Addon for Decentralized Federated AI

Integrate [FLock FL Alliance](https://github.com/FLock-io/FL-Alliance-Client) (`FLockAlliance`) with Open Cluster Management (OCM) to automate decentralized federated learning across multi-cluster and multi-cloud environments, with blockchain-backed coordination and incentive mechanisms for participating nodes.

## Key Characteristics

- Runtime mode is fixed to `local`
- `FLocKit` is not deployed as a separate addon workload
- Each managed cluster runs one `flock-agent` Deployment in `flock-system`
- Runtime configuration is loaded from a mounted node directory, usually `/data/flock-client`

## Features

| Capability | Description |
| --- | --- |
| Deployment | Declarative addon rollout from the hub through `ClusterManagementAddOn`, `AddOnTemplate`, and `AddOnDeploymentConfig` |
| Scheduling | Automatic GPU/CPU template selection based on the managed cluster label `gpu=true` |
| Runtime | One direct `flock-alliance-client` workload per managed cluster with node-mounted `.env` and local data |
| Modes | Support for testnet, local chain + original S3, and local chain + local S3-compatible storage |

## Supported Deployment Modes

| Mode                              | Command | Best for |
|-----------------------------------| --- | --- |
| Testnet                           | `make deploy-testnet` | Standard first deployment against an existing blockchain endpoint |
| Local chain + original S3         | `make deploy-local-chain-s3` | Hub-hosted local chain while reusing an existing uploaded model hash |
| Local chain + local S3-compatible | `make deploy-local-chain-s3-compatible` | Fully local hub-side chain and object storage for development |

Full mode details are in [Deployment Modes](docs/deployment-modes.md).

## Architecture

```text
┌─────────────────────────────────────────────────────────────┐
│                        Hub Cluster                          │
│  ┌────────────────────────────────────────────────────────┐ │
│  │  ClusterManagementAddOn + AddOnTemplate               │ │
│  │  AddOnDeploymentConfig                                │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                     Managed Cluster                         │
│  namespace: flock-system                                   │
│  Deployment: flock-agent                                   │
│  Container: flock-alliance-client                          │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                  Managed Cluster Node                       │
│  hostPath: /data/flock-client                              │
│  files: .env, datasets, model inputs                       │
└─────────────────────────────────────────────────────────────┘
```

## Documentation

- [Prepare Multi-Cluster Environment](docs/prepare-multicluster-environment.md) - build Kubernetes clusters, install OCM, register managed clusters, and verify ManifestWork distribution
- [Install FLock Addon](docs/install-flock-addon.md) - first deployment path for the default testnet workflow
- [Deployment Modes](docs/deployment-modes.md) - compare and run the three supported deployment modes
- [Image Management](docs/image-management.md) - choose public/private images and publish custom builds
- [Configuration and Overrides](docs/configuration-and-overrides.md) - runtime model, path rules, task updates, and per-cluster overrides
- [Troubleshooting](docs/troubleshooting.md) - image pull, OCM distribution, GPU mapping, and log collection

## Related Projects

- [FLockAlliance](https://github.com/FLock-io/FL-Alliance-Client) - the direct client runtime deployed by this addon
- [Open Cluster Management](https://open-cluster-management.io) - multi-cluster management for Kubernetes
