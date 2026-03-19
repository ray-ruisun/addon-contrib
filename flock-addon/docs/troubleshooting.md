# FLock Addon Troubleshooting

This guide covers the most common failure modes for `flock-addon`.

## 1) Pod Stuck in `ImagePullBackOff` or `ErrImagePull`

Inspect the Pod events first.

```bash
# [Managed Cluster context]
kubectl -n flock-system describe pod -l app.kubernetes.io/name=flock-addon
```

Check the image value from the Hub:

```bash
# [Hub]
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config -o yaml | rg -n "FLOCK_ALLIANCE_IMAGE|value"
```

Should see:

- the Pod events show the exact pull failure
- `FLOCK_ALLIANCE_IMAGE` matches the intended image

Also confirm the image was actually published:

```bash
# [FL-Alliance-Client workspace]
export IMAGE_SHA=$(git rev-parse --short=12 HEAD)
make image-inspect IMAGE_OWNER='ray-ruisun' IMAGE_TAG='latest' IMAGE_IMMUTABLE_TAG="$IMAGE_SHA"
```

Should see:

- both `latest` and `$IMAGE_SHA` exist locally before you expect the cluster to pull them
- the addon should deploy `$IMAGE_SHA`

If the image is wrong:

```bash
# [Hub]
IMAGE_OWNER='ray-ruisun' IMAGE_TAG="$IMAGE_SHA" IMAGE_PULL_POLICY='Always' make deploy-testnet TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
make disable-addon CLUSTER=cluster1
make enable-addon CLUSTER=cluster1
```

## 2) Registry Returns `unauthorized` or `denied`

This usually means you are following the private repository path, but the managed cluster does not have a valid pull secret.

Create an image pull secret on the managed cluster:

```bash
# [Managed Cluster context]
kubectl -n flock-system create secret docker-registry ghcr-creds \
  --docker-server=ghcr.io \
  --docker-username='<github-user>' \
  --docker-password='<github-token>' \
  --docker-email='<email>'
```

Redeploy from the Hub:

```bash
# [Hub]
helm upgrade --install flock-addon charts/flock-addon \
  --set image.repository='ghcr.io/ray-ruisun/fl-alliance-client' \
  --set image.tag="$IMAGE_SHA" \
  --set image.pullSecrets[0]='ghcr-creds'
```

Check:

```bash
# [Managed Cluster context]
kubectl -n flock-system get secret ghcr-creds
kubectl -n flock-system get deploy flock-agent -o yaml | rg -n "imagePullSecrets|ghcr-creds"
```

Should see:

- `ghcr-creds` exists
- `deployment/flock-agent` references `imagePullSecrets`

## 3) Hub Deploy Succeeds but Nothing Runs on Managed Cluster

Check the OCM distribution chain on the Hub:

```bash
# [Hub]
kubectl get clustermanagementaddon flock-addon
kubectl -n cluster1 get managedclusteraddon flock-addon -o yaml
kubectl -n cluster1 get manifestwork
```

Should see:

- `clustermanagementaddon/flock-addon` exists
- `managedclusteraddon/flock-addon` exists in the managed cluster namespace on the Hub
- a `ManifestWork` exists

If `ManagedClusterAddOn` exists but no `ManifestWork` appears:

```bash
# [Hub]
make disable-addon CLUSTER=cluster1
make enable-addon CLUSTER=cluster1
```

Check again:

```bash
# [Hub]
kubectl -n cluster1 get managedclusteraddon flock-addon -o yaml
kubectl -n cluster1 get manifestwork
```

Should see:

- `spec.configs` contains both `flock-addon` and `flock-addon-config`
- `ManifestWork` appears after re-enable

## 4) Namespace Exists but Pod Does Not Start

Run on the managed cluster:

```bash
# [Managed Cluster context]
kubectl get ns flock-system
kubectl -n flock-system get deploy,pod
kubectl -n flock-system describe deploy flock-agent
```

Should see:

- namespace `flock-system` exists
- `flock-agent` Deployment exists
- Deployment events explain whether the problem is image, mount, or scheduling

## 5) `.env` Is Not Being Loaded

Confirm the node path and container path mapping:

```bash
# [Managed Cluster context]
kubectl -n flock-system get deploy flock-agent -o yaml | rg -n "HOST_DATA_PATH|FLOCK_ALLIANCE_ENV_FILE|mountPath|hostPath"
```

Should see:

- host path points to `/data/flock-client`
- container mount path is `/data`
- env file path is `/data/.env`

## 6) Training Is Slow or GPU Seems Missing

First confirm the Pod actually requests GPU resources and lands on a GPU-capable node.

```bash
# [Managed Cluster context]
kubectl -n flock-system get pod -l app.kubernetes.io/name=flock-addon -o wide
kubectl -n flock-system describe pod -l app.kubernetes.io/name=flock-addon | rg -n "nvidia.com/gpu|Node:|FailedScheduling|Warning"
kubectl get node -o custom-columns=NAME:.metadata.name,GPU_ALLOCATABLE:.status.allocatable.nvidia\\.com/gpu
kubectl get ds -A | rg -i "nvidia|gpu|device-plugin"
kubectl -n flock-system logs deploy/flock-agent -c flock-alliance-client --tail=80 | rg -n "NVIDIA device files|nvidia-smi|No NVIDIA device files"
```

Should see:

- container resources include `nvidia.com/gpu`
- Pod is scheduled on a node with non-zero `GPU_ALLOCATABLE`
- no `FailedScheduling` events caused by missing GPU resources
- GPU device plugin DaemonSet exists and is Ready
- startup logs show whether `/dev/nvidia*` exists inside the container

Then inspect the FLocKit subprocess log for runtime device detection:

```bash
# [Managed Cluster context]
POD=$(kubectl -n flock-system get pod -l app.kubernetes.io/component=agent -o jsonpath='{.items[0].metadata.name}')
kubectl -n flock-system exec "$POD" -c flock-alliance-client -- sh -lc 'f=$(ls -1t /app/output/task_outputs/process_*.log | head -n1); echo "LOG=$f"; rg -n "CUDA is available|CUDA not available|Using device/backend|device=" "$f" || true'
```

Interpretation:

- If you see `CUDA is available`, GPU mapping is working.
- If you see `CUDA not available`, the process is running CPU fallback.

If you want CPU mode intentionally, redeploy with:

```bash
# [Hub]
USE_GPU='false' GPU_RESOURCE_ENABLED='false' make deploy-testnet TASK_ADDRESS='0x...'
```

If GPU nodes are tainted or use dedicated labels, deploy with matching scheduling hints:

```bash
# [Hub]
helm upgrade --install flock-addon charts/flock-addon \
  --set agent.nodeSelector.gpu=true \
  --set 'agent.tolerations[0].key=nvidia.com/gpu' \
  --set 'agent.tolerations[0].operator=Exists' \
  --set 'agent.tolerations[0].effect=NoSchedule'
```
