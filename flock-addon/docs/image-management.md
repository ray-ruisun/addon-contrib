# Image Management

Use this guide to choose the runtime image for `flock-addon`, publish a custom build, and verify the managed cluster is pulling the expected image.

## Default Image Resolution

Chart fallback image:

- `ghcr.io/flock-io/fl-alliance-client:<release-tag>`

Environment-variable based overrides supported by the `Makefile`:

- `IMAGE_REGISTRY`, default `ghcr.io`
- `IMAGE_OWNER`, default `flock-io`
- `IMAGE_NAME`, default `fl-alliance-client`
- `IMAGE_TAG`, default `latest`
- `IMAGE_PULL_POLICY`, default `Always`
- `IMAGE_PULL_SECRET`, optional managed-cluster image pull secret name
- `FLOCK_ALLIANCE_IMAGE`, overrides all of the above

Example:

```bash
# [Hub]
export IMAGE_REGISTRY='ghcr.io'
export IMAGE_OWNER='ray-ruisun'
export IMAGE_NAME='fl-alliance-client'
export IMAGE_TAG='<git-sha-or-release-tag>'
export IMAGE_PULL_POLICY='Always'
export FLOCK_ALLIANCE_IMAGE="${IMAGE_REGISTRY}/${IMAGE_OWNER}/${IMAGE_NAME}:${IMAGE_TAG}"
```

Use it for deployment:

```bash
# [Hub]
make deploy-testnet TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
```

Check:

```bash
# [Hub]
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config -o yaml | rg -n "FLOCK_ALLIANCE_IMAGE|value"
```

Should see:

- `FLOCK_ALLIANCE_IMAGE` matches the image you intend to deploy

## Public Image Repository

Use this path when the image already exists publicly in GHCR, for example:

- `ghcr.io/flock-io/fl-alliance-client:<release-tag>`

How to operate:

- do not clone `FL-Alliance-Client` unless you want to rebuild the image
- do not create an image pull secret
- deploy directly from `flock-addon`

Example:

```bash
# [Hub]
unset IMAGE_PULL_SECRET
export IMAGE_OWNER='flock-io'
export IMAGE_TAG='<release-tag>'
make deploy-testnet TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
```

## Private Image Repository

Use this path when the image is private, for example:

- `ghcr.io/your-org/fl-alliance-client:<git-sha-or-release-tag>`

How to operate:

1. Publish the image before addon deployment.
2. Create a pull secret on every managed cluster.
3. Set `IMAGE_PULL_SECRET` on the hub before deploy.

Create the registry secret on each managed cluster:

```bash
# [Managed Cluster context]
kubectl -n flock-system create secret docker-registry ghcr-pull \
  --docker-server=ghcr.io \
  --docker-username="$GHCR_USER" \
  --docker-password="$GHCR_PAT" \
  --dry-run=client -o yaml | kubectl apply -f -
```

Deploy from the hub:

```bash
# [Hub]
export IMAGE_OWNER='your-org'
export IMAGE_TAG='<git-sha-or-release-tag>'
export IMAGE_PULL_SECRET='ghcr-pull'
export IMAGE_PULL_POLICY='Always'
make deploy-testnet TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
```

Check:

```bash
# [Managed Cluster context]
kubectl -n flock-system get secret ghcr-pull
kubectl -n flock-system get deploy flock-agent -o yaml | rg -n "imagePullSecrets|ghcr-pull"
```

Should see:

- `ghcr-pull` exists
- `deployment/flock-agent` references `imagePullSecrets`

## Publish a Custom Image

You only need the `FL-Alliance-Client` source repository if you want to build or publish your own image.

Source repository:

- [FL-Alliance-Client](https://github.com/FLock-io/FL-Alliance-Client.git)

Clone it before local image build or manual image push:

```bash
# [Hub or image-build machine]
cd ~
git clone https://github.com/FLock-io/FL-Alliance-Client.git
cd FL-Alliance-Client
```

Example publish flow:

```bash
# [FL-Alliance-Client workspace]
export IMAGE_SHA=$(git rev-parse --short=12 HEAD)
echo "$GHCR_PAT" | docker login ghcr.io -u "$GHCR_USER" --password-stdin
make image-publish \
  DOCKER='sudo docker' \
  IMAGE_OWNER="$GHCR_USER" \
  IMAGE_TAG="$IMAGE_SHA" \
  IMAGE_IMMUTABLE_TAG="$IMAGE_SHA" \
  GHCR_USER="$GHCR_USER" \
  GHCR_PAT="$GHCR_PAT"
```

If your environment does not need `sudo` for Docker, you can omit `DOCKER='sudo docker'`.

Check:

```bash
# [Hub]
export IMAGE_OWNER="$GHCR_USER"
export IMAGE_TAG="$IMAGE_SHA"
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config -o yaml | rg -n "FLOCK_ALLIANCE_IMAGE|value"
```

Should see:

- addon deployment uses an immutable tag such as `$IMAGE_SHA`

If you use GitHub Actions instead of local push, wait for the publish workflow to finish successfully before deploying the addon.

## Verify the Deployed Image

```bash
# [Hub]
kubectl -n open-cluster-management get addondeploymentconfig flock-addon-config -o yaml | rg -n "FLOCK_ALLIANCE_IMAGE|value"

# [Managed Cluster context]
kubectl -n flock-system describe pod -l app.kubernetes.io/name=flock-addon
```

If you republish the same tag, redeploy with an explicit image override and re-enable the addon:

```bash
# [Hub]
IMAGE_OWNER='ray-ruisun' IMAGE_TAG='<git-sha-or-release-tag>' IMAGE_PULL_POLICY='Always' make deploy-testnet TASK_ADDRESS='0x47B0397C6ae306002788D093b29bcD2EDAd19924'
make disable-addon CLUSTER=cluster1
make enable-addon CLUSTER=cluster1
```

If you also rotated credentials for a private registry, recreate the secret and restart the addon Pods:

```bash
# [Managed Cluster context]
kubectl -n flock-system delete secret ghcr-pull --ignore-not-found
kubectl -n flock-system create secret docker-registry ghcr-pull \
  --docker-server=ghcr.io \
  --docker-username="$GHCR_USER" \
  --docker-password="$GHCR_PAT" \
  --dry-run=client -o yaml | kubectl apply -f -
kubectl -n flock-system delete pod --all
kubectl -n flock-system get pod -w
```
