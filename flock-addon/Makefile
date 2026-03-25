KUBECTL ?= kubectl
HELM ?= helm
NAMESPACE ?= open-cluster-management
RELEASE_NAME ?= flock-addon
CHART_PATH ?= charts/flock-addon

IMAGE_REGISTRY ?= ghcr.io
IMAGE_OWNER ?= flock-io
IMAGE_NAME ?= fl-alliance-client
IMAGE_TAG ?= latest
IMAGE_PULL_POLICY ?= Always
USE_GPU ?= true
GPU_RESOURCE_ENABLED ?= $(USE_GPU)
FLOCK_ALLIANCE_IMAGE ?= $(IMAGE_REGISTRY)/$(IMAGE_OWNER)/$(IMAGE_NAME):$(IMAGE_TAG)
IMAGE_PULL_SECRET ?=

ifneq ($(strip $(IMAGE_PULL_SECRET)),)
IMAGE_PULL_SECRET_ARG := --set image.pullSecrets[0]='$(IMAGE_PULL_SECRET)'
endif

ifeq ($(strip $(IMAGE_TAG)),)
$(error IMAGE_TAG is empty. Set IMAGE_TAG to a git SHA or release tag before deploy, for example: export IMAGE_SHA=$$(git rev-parse --short=12 HEAD) && export IMAGE_TAG="$$IMAGE_SHA")
endif

.PHONY: all
all: help

##@ Deployment

.PHONY: deploy
deploy: ## Deploy FLock addon via Helm
	$(HELM) upgrade --install $(RELEASE_NAME) $(CHART_PATH) \
		--set addon.installStrategy=Manual \
		--set placement.gpu.enabled=false \
		--set placement.all.enabled=false \
		--set deploymentConfig.runtime.useGpu='$(USE_GPU)' \
		--set agent.gpuResource.enabled=$(GPU_RESOURCE_ENABLED) \
		--set image.repository=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f1) \
		--set image.tag=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f2) \
		--set image.pullPolicy='$(IMAGE_PULL_POLICY)' \
		$(IMAGE_PULL_SECRET_ARG)

.PHONY: upgrade
upgrade: ## Upgrade FLock addon with current image/runtime values
	$(HELM) upgrade $(RELEASE_NAME) $(CHART_PATH) \
		--set deploymentConfig.runtime.useGpu='$(USE_GPU)' \
		--set agent.gpuResource.enabled=$(GPU_RESOURCE_ENABLED) \
		--set image.repository=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f1) \
		--set image.tag=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f2) \
		--set image.pullPolicy='$(IMAGE_PULL_POLICY)' \
		$(IMAGE_PULL_SECRET_ARG) \
		--set deploymentConfig.blockchain.rpc='$(RPC)' \
		--set deploymentConfig.blockchain.tokenAddress='$(TOKEN_ADDRESS)' \
		--set deploymentConfig.blockchain.taskAddress='$(TASK_ADDRESS)'

.PHONY: deploy-testnet
deploy-testnet: ## Deploy for testnet mode (usage: make deploy-testnet TASK_ADDRESS=0x... [RPC=https://...] [TOKEN_ADDRESS=0x...])
ifndef TASK_ADDRESS
	$(error TASK_ADDRESS is not set. Usage: make deploy-testnet TASK_ADDRESS=0x... [RPC=https://...] [TOKEN_ADDRESS=0x...])
endif
	$(HELM) upgrade --install $(RELEASE_NAME) $(CHART_PATH) \
		--set addon.installStrategy=Manual \
		--set placement.gpu.enabled=false \
		--set placement.all.enabled=false \
		--set deploymentConfig.runtime.useGpu='$(USE_GPU)' \
		--set agent.gpuResource.enabled=$(GPU_RESOURCE_ENABLED) \
		--set image.repository=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f1) \
		--set image.tag=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f2) \
		--set image.pullPolicy='$(IMAGE_PULL_POLICY)' \
		$(IMAGE_PULL_SECRET_ARG) \
		--set deploymentConfig.blockchain.rpc='$(RPC)' \
		--set deploymentConfig.blockchain.tokenAddress='$(TOKEN_ADDRESS)' \
		--set deploymentConfig.blockchain.taskAddress='$(TASK_ADDRESS)' \
		--set deploymentConfig.storage.backend='s3'

.PHONY: update-task
update-task: ## Update shared task address only (usage: make update-task TASK_ADDRESS=0x...)
ifndef TASK_ADDRESS
	$(error TASK_ADDRESS is not set. Usage: make update-task TASK_ADDRESS=0x...)
endif
	$(HELM) upgrade $(RELEASE_NAME) $(CHART_PATH) \
		--reuse-values \
		--set deploymentConfig.blockchain.taskAddress='$(TASK_ADDRESS)'

.PHONY: undeploy
undeploy: ## Remove FLock addon release
	$(HELM) uninstall $(RELEASE_NAME) || true

.PHONY: deploy-auto-gpu
deploy-auto-gpu: ## Deploy with auto-install for gpu=true clusters ([TASK_ADDRESS=0x...] [RPC=https://...] [TOKEN_ADDRESS=0x...])
	$(HELM) upgrade --install $(RELEASE_NAME) $(CHART_PATH) \
		--set addon.installStrategy=Placements \
		--set placement.gpu.enabled=true \
		--set placement.all.enabled=false \
		--set deploymentConfig.runtime.useGpu='$(USE_GPU)' \
		--set placement.gpu.config.useGpu='$(USE_GPU)' \
		--set agent.gpuResource.enabled=$(GPU_RESOURCE_ENABLED) \
		--set image.repository=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f1) \
		--set image.tag=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f2) \
		--set image.pullPolicy='$(IMAGE_PULL_POLICY)' \
		$(IMAGE_PULL_SECRET_ARG) \
		--set deploymentConfig.blockchain.rpc='$(RPC)' \
		--set deploymentConfig.blockchain.tokenAddress='$(TOKEN_ADDRESS)' \
		--set deploymentConfig.blockchain.taskAddress='$(TASK_ADDRESS)'

.PHONY: deploy-auto-all
deploy-auto-all: ## Deploy with auto-install for all clusters in global set ([TASK_ADDRESS=0x...] [RPC=https://...] [TOKEN_ADDRESS=0x...])
	$(HELM) upgrade --install $(RELEASE_NAME) $(CHART_PATH) \
		--set addon.installStrategy=Placements \
		--set placement.gpu.enabled=false \
		--set placement.all.enabled=true \
		--set deploymentConfig.runtime.useGpu='$(USE_GPU)' \
		--set placement.all.config.useGpu='$(USE_GPU)' \
		--set agent.gpuResource.enabled=$(GPU_RESOURCE_ENABLED) \
		--set image.repository=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f1) \
		--set image.tag=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f2) \
		--set image.pullPolicy='$(IMAGE_PULL_POLICY)' \
		$(IMAGE_PULL_SECRET_ARG) \
		--set deploymentConfig.blockchain.rpc='$(RPC)' \
		--set deploymentConfig.blockchain.tokenAddress='$(TOKEN_ADDRESS)' \
		--set deploymentConfig.blockchain.taskAddress='$(TASK_ADDRESS)'

.PHONY: enable-addon
enable-addon: ## Enable addon on a cluster (usage: make enable-addon CLUSTER=cluster1 [CONFIG=flock-addon-config])
ifndef CLUSTER
	$(error CLUSTER is not set. Usage: make enable-addon CLUSTER=cluster1 [CONFIG=flock-addon-config])
endif
	@echo "apiVersion: addon.open-cluster-management.io/v1alpha1" | \
	{ cat; echo "kind: ManagedClusterAddOn"; } | \
	{ cat; echo "metadata:"; } | \
	{ cat; echo "  name: flock-addon"; } | \
	{ cat; echo "  namespace: $(CLUSTER)"; } | \
	{ cat; echo "  annotations:"; } | \
	{ cat; echo "    addon.open-cluster-management.io/v1alpha1-install-namespace: flock-system"; } | \
	{ cat; echo "spec:"; } | \
	{ cat; echo "  configs:"; } | \
	{ cat; echo "    - group: addon.open-cluster-management.io"; } | \
	{ cat; echo "      resource: addontemplates"; } | \
	{ cat; echo "      name: flock-addon"; } | \
	{ cat; echo "    - group: addon.open-cluster-management.io"; } | \
	{ cat; echo "      resource: addondeploymentconfigs"; } | \
	{ cat; echo "      name: $${CONFIG:-flock-addon-config}"; } | \
	{ cat; echo "      namespace: $(NAMESPACE)"; } | \
	$(KUBECTL) apply -f -

.PHONY: disable-addon
disable-addon: ## Disable addon on a cluster (usage: make disable-addon CLUSTER=cluster1)
ifndef CLUSTER
	$(error CLUSTER is not set. Usage: make disable-addon CLUSTER=cluster1)
endif
	$(KUBECTL) delete managedclusteraddon flock-addon -n $(CLUSTER) --ignore-not-found

.PHONY: status
status: ## Show addon objects
	$(KUBECTL) get clustermanagementaddons
	$(KUBECTL) get addontemplates
	$(KUBECTL) get addondeploymentconfigs -A
	$(KUBECTL) get managedclusteraddons -A

##@ CI Targets

.PHONY: verify
verify: lint ## Verify chart and templates
	@echo "Verification complete"

.PHONY: vendor
vendor: ## No-op vendor target for non-Go addon
	@echo "No vendor step required"

.PHONY: build
build: ## No-op build target for Helm-only addon
	@echo "No build required"

.PHONY: test-unit
test-unit: ## No-op unit tests for Helm-only addon
	@echo "No unit tests"

.PHONY: test-integration
test-integration: ## No-op integration tests for Helm-only addon
	@echo "No integration tests"

.PHONY: test-e2e
test-e2e: ## No-op e2e tests for Helm-only addon
	@echo "No e2e tests"

.PHONY: test-chart
test-chart: ## Helm chart validation used by CI
	$(HELM) lint $(CHART_PATH)

.PHONY: image
image: ## No-op image target; images are built in source repositories
	@echo "Use source repositories to build images"

.PHONY: image-push
image-push: ## No-op image push target
	@echo "Use source repositories to push images"

.PHONY: image-manifest
image-manifest: ## No-op multi-arch manifest target
	@echo "Use source repositories to publish manifests"

##@ Development

.PHONY: lint
lint: ## Lint the Helm chart
	$(HELM) lint $(CHART_PATH)

.PHONY: template
template: ## Render templates locally
	$(HELM) template $(RELEASE_NAME) $(CHART_PATH)

##@ Help

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-25s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
