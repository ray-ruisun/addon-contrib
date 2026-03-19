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

If the image is wrong:

```bash
# [Hub]
IMAGE_OWNER='ray-ruisun' IMAGE_TAG='v0.1.0' make deploy-testnet TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
make disable-addon CLUSTER=cluster1
make enable-addon CLUSTER=cluster1
```

## 2) Registry Returns `unauthorized` or `denied`

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
