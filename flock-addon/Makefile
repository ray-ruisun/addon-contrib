KUBECTL ?= kubectl
HELM ?= helm
NAMESPACE ?= open-cluster-management
RELEASE_NAME ?= flock-addon
CHART_PATH ?= charts/flock-addon

FLOCK_ALLIANCE_IMAGE ?= ghcr.io/flock-io/fl-alliance-client:v0.1.0

.PHONY: all
all: help

##@ Deployment

.PHONY: deploy
deploy: ## Deploy FLock addon via Helm
	$(HELM) upgrade --install $(RELEASE_NAME) $(CHART_PATH) \
		--set image.repository=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f1) \
		--set image.tag=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f2)

.PHONY: deploy-testnet
deploy-testnet: ## Deploy for shared testnet mode (usage: make deploy-testnet RPC=... TOKEN_ADDRESS=... TASK_ADDRESS=...)
ifndef RPC
	$(error RPC is not set. Usage: make deploy-testnet RPC=https://... TOKEN_ADDRESS=0x... TASK_ADDRESS=0x...)
endif
ifndef TOKEN_ADDRESS
	$(error TOKEN_ADDRESS is not set. Usage: make deploy-testnet RPC=https://... TOKEN_ADDRESS=0x... TASK_ADDRESS=0x...)
endif
ifndef TASK_ADDRESS
	$(error TASK_ADDRESS is not set. Usage: make deploy-testnet RPC=https://... TOKEN_ADDRESS=0x... TASK_ADDRESS=0x...)
endif
	$(HELM) upgrade --install $(RELEASE_NAME) $(CHART_PATH) \
		--set image.repository=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f1) \
		--set image.tag=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f2) \
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
		--set deploymentConfig.blockchain.taskAddress='$(TASK_ADDRESS)' \
		--set deploymentConfig.storage.backend='s3'

.PHONY: undeploy
undeploy: ## Remove FLock addon release
	$(HELM) uninstall $(RELEASE_NAME) || true

.PHONY: deploy-auto-gpu
deploy-auto-gpu: ## Deploy with auto-install for gpu=true clusters
	$(HELM) upgrade --install $(RELEASE_NAME) $(CHART_PATH) \
		--set addon.installStrategy=Placements \
		--set placement.gpu.enabled=true

.PHONY: deploy-auto-all
deploy-auto-all: ## Deploy with auto-install for all clusters in global set
	$(HELM) upgrade --install $(RELEASE_NAME) $(CHART_PATH) \
		--set addon.installStrategy=Placements \
		--set placement.all.enabled=true

.PHONY: enable-addon
enable-addon: ## Enable addon on a cluster (usage: make enable-addon CLUSTER=cluster1)
ifndef CLUSTER
	$(error CLUSTER is not set. Usage: make enable-addon CLUSTER=cluster1)
endif
	@echo "apiVersion: addon.open-cluster-management.io/v1alpha1" | \
	{ cat; echo "kind: ManagedClusterAddOn"; } | \
	{ cat; echo "metadata:"; } | \
	{ cat; echo "  name: flock-addon"; } | \
	{ cat; echo "  namespace: $(CLUSTER)"; } | \
	{ cat; echo "spec:"; } | \
	{ cat; echo "  installNamespace: flock-system"; } | \
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
