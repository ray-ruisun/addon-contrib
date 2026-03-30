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
FLOCK_ALLIANCE_IMAGE ?= $(IMAGE_REGISTRY)/$(IMAGE_OWNER)/$(IMAGE_NAME):$(IMAGE_TAG)
IMAGE_PULL_SECRET ?=
HOST_DATA_PATH ?= /data/flock-client
LOCAL_STORAGE_DIR ?= /data/shared
FL_ALLIANCE_CLIENT_DIR ?=
MODEL_ARCHIVE ?=
RPC_HOST ?=
HUB_SHARED_MODELS_DIR ?=
CHAIN_DOCKER_COMPOSE ?=

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
		--set image.repository=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f1) \
		--set image.tag=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f2) \
		--set image.pullPolicy='$(IMAGE_PULL_POLICY)' \
		$(IMAGE_PULL_SECRET_ARG)

.PHONY: upgrade
upgrade: ## Upgrade FLock addon with current image/runtime values
	$(HELM) upgrade $(RELEASE_NAME) $(CHART_PATH) \
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
		--set image.repository=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f1) \
		--set image.tag=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f2) \
		--set image.pullPolicy='$(IMAGE_PULL_POLICY)' \
		$(IMAGE_PULL_SECRET_ARG) \
		--set deploymentConfig.blockchain.rpc='$(RPC)' \
		--set deploymentConfig.blockchain.tokenAddress='$(TOKEN_ADDRESS)' \
		--set deploymentConfig.blockchain.taskAddress='$(TASK_ADDRESS)' \
		--set deploymentConfig.storage.backend='s3'

.PHONY: deploy-local-chain
deploy-local-chain: ## Deploy for hub-hosted local chain + shared local storage (usage: make deploy-local-chain RPC=http://<hub-ip>:8545 [TOKEN_ADDRESS=0x...] [TASK_ADDRESS=0x...])
ifndef RPC
	$(error RPC is not set. Usage: make deploy-local-chain RPC=http://<hub-ip>:8545 [TOKEN_ADDRESS=0x...] [TASK_ADDRESS=0x...])
endif
	$(HELM) upgrade --install $(RELEASE_NAME) $(CHART_PATH) \
		--set addon.installStrategy=Manual \
		--set placement.gpu.enabled=false \
		--set placement.all.enabled=false \
		--set image.repository=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f1) \
		--set image.tag=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f2) \
		--set image.pullPolicy='$(IMAGE_PULL_POLICY)' \
		$(IMAGE_PULL_SECRET_ARG) \
		--set agent.dataVolume.hostPath='$(HOST_DATA_PATH)' \
		--set deploymentConfig.blockchain.rpc='$(RPC)' \
		--set deploymentConfig.blockchain.tokenAddress='$(TOKEN_ADDRESS)' \
		--set deploymentConfig.blockchain.taskAddress='$(TASK_ADDRESS)' \
		--set deploymentConfig.training.noIncentive='false' \
		--set deploymentConfig.storage.backend='local' \
		--set deploymentConfig.storage.localSharedDir='$(LOCAL_STORAGE_DIR)'

.PHONY: deploy-local-stack
deploy-local-stack: ## Start local chain on the Hub via FL-Alliance-Client, then deploy flock-addon (requires FL_ALLIANCE_CLIENT_DIR, MODEL_ARCHIVE, RPC_HOST)
ifndef FL_ALLIANCE_CLIENT_DIR
	$(error FL_ALLIANCE_CLIENT_DIR is not set. Usage: make deploy-local-stack FL_ALLIANCE_CLIENT_DIR=/path/to/FL-Alliance-Client MODEL_ARCHIVE=/path/to/model.tar.gz RPC_HOST=<hub-ip>)
endif
ifndef MODEL_ARCHIVE
	$(error MODEL_ARCHIVE is not set. Usage: make deploy-local-stack FL_ALLIANCE_CLIENT_DIR=/path/to/FL-Alliance-Client MODEL_ARCHIVE=/path/to/model.tar.gz RPC_HOST=<hub-ip>)
endif
ifndef RPC_HOST
	$(error RPC_HOST is not set. Usage: make deploy-local-stack FL_ALLIANCE_CLIENT_DIR=/path/to/FL-Alliance-Client MODEL_ARCHIVE=/path/to/model.tar.gz RPC_HOST=<hub-ip>)
endif
	@set -e; \
	MODEL_HASH=$$(sha256sum '$(MODEL_ARCHIVE)' | cut -d' ' -f1); \
	echo "MODEL_HASH=$$MODEL_HASH"; \
	if [ -n "$(HUB_SHARED_MODELS_DIR)" ]; then \
		mkdir -p '$(HUB_SHARED_MODELS_DIR)'; \
		cp '$(MODEL_ARCHIVE)' '$(HUB_SHARED_MODELS_DIR)'/"$$MODEL_HASH"; \
		echo "Copied model archive to $(HUB_SHARED_MODELS_DIR)/$$MODEL_HASH"; \
	else \
		echo "HUB_SHARED_MODELS_DIR is empty; skipping model archive copy."; \
		echo "Make sure the shared storage already contains models/$$MODEL_HASH before enabling the addon."; \
	fi; \
	$(MAKE) -C '$(FL_ALLIANCE_CLIENT_DIR)' chain \
		$(if $(strip $(CHAIN_DOCKER_COMPOSE)),DOCKER_COMPOSE='$(CHAIN_DOCKER_COMPOSE)') \
		MODEL_DEFINITION_HASH="$$MODEL_HASH" \
		$(if $(strip $(ROUNDS)),ROUNDS='$(ROUNDS)') \
		$(if $(strip $(MIN_PARTICIPANTS)),MIN_PARTICIPANTS='$(MIN_PARTICIPANTS)') \
		$(if $(strip $(TIME_TO_PROPOSE)),TIME_TO_PROPOSE='$(TIME_TO_PROPOSE)') \
		$(if $(strip $(TIME_TO_VOTE)),TIME_TO_VOTE='$(TIME_TO_VOTE)'); \
	TOKEN_ADDRESS=$$(python3 -c "import json; print(json.load(open('$(FL_ALLIANCE_CLIENT_DIR)/data/contracts.json'))['FlockToken']['address'])"); \
	TASK_ADDRESS=$$(python3 -c "import json; print(json.load(open('$(FL_ALLIANCE_CLIENT_DIR)/data/contracts.json'))['FlockTask']['address'])"); \
	echo "TOKEN_ADDRESS=$$TOKEN_ADDRESS"; \
	echo "TASK_ADDRESS=$$TASK_ADDRESS"; \
	$(MAKE) deploy-local-chain \
		RPC='http://$(RPC_HOST):8545' \
		TOKEN_ADDRESS="$$TOKEN_ADDRESS" \
		TASK_ADDRESS="$$TASK_ADDRESS"

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
		--set image.repository=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f1) \
		--set image.tag=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f2) \
		--set image.pullPolicy='$(IMAGE_PULL_POLICY)' \
		$(IMAGE_PULL_SECRET_ARG) \
		--set deploymentConfig.blockchain.rpc='$(RPC)' \
		--set deploymentConfig.blockchain.tokenAddress='$(TOKEN_ADDRESS)' \
		--set deploymentConfig.blockchain.taskAddress='$(TASK_ADDRESS)'

.PHONY: deploy-auto-all
deploy-auto-all: ## Deploy with dynamic auto-install: gpu=true clusters use GPU template, others use CPU template
	$(HELM) upgrade --install $(RELEASE_NAME) $(CHART_PATH) \
		--set addon.installStrategy=Placements \
		--set placement.gpu.enabled=true \
		--set placement.all.enabled=true \
		--set image.repository=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f1) \
		--set image.tag=$$(echo $(FLOCK_ALLIANCE_IMAGE) | cut -d: -f2) \
		--set image.pullPolicy='$(IMAGE_PULL_POLICY)' \
		$(IMAGE_PULL_SECRET_ARG) \
		--set deploymentConfig.blockchain.rpc='$(RPC)' \
		--set deploymentConfig.blockchain.tokenAddress='$(TOKEN_ADDRESS)' \
		--set deploymentConfig.blockchain.taskAddress='$(TASK_ADDRESS)'

.PHONY: enable-addon
enable-addon: ## Enable addon on a cluster; auto-select gpu/cpu template from managedcluster label gpu=true (override with CONFIG=/TEMPLATE=)
ifndef CLUSTER
	$(error CLUSTER is not set. Usage: make enable-addon CLUSTER=cluster1 [CONFIG=flock-addon-config] [TEMPLATE=flock-addon])
endif
	@GPU_LABEL=$$($(KUBECTL) get managedcluster $(CLUSTER) -o jsonpath='{.metadata.labels.gpu}' 2>/dev/null || true); \
	if [ -n "$(TEMPLATE)" ]; then \
		TEMPLATE_NAME='$(TEMPLATE)'; \
	elif [ "$$GPU_LABEL" = "true" ]; then \
		TEMPLATE_NAME='$(RELEASE_NAME)-gpu'; \
	else \
		TEMPLATE_NAME='$(RELEASE_NAME)'; \
	fi; \
	if [ -n "$(CONFIG)" ]; then \
		CONFIG_NAME='$(CONFIG)'; \
	elif [ "$$GPU_LABEL" = "true" ]; then \
		CONFIG_NAME='$(RELEASE_NAME)-gpu-config'; \
	else \
		CONFIG_NAME='$(RELEASE_NAME)-config'; \
	fi; \
	echo "apiVersion: addon.open-cluster-management.io/v1alpha1" | \
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
	{ cat; echo "      name: $$TEMPLATE_NAME"; } | \
	{ cat; echo "    - group: addon.open-cluster-management.io"; } | \
	{ cat; echo "      resource: addondeploymentconfigs"; } | \
	{ cat; echo "      name: $$CONFIG_NAME"; } | \
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
