# -----------------------------------------------------------------------------
# flock-addon Makefile
#
# Drives the addon's lifecycle and CI gates without depending on a hub-side
# operator. Targets are grouped into sections that `make help` prints:
#
#   Deployment   - public `make deploy-*`, `make update-task`, `make undeploy`,
#                  `make enable-addon`, `make disable-addon`
#   CI Targets   - contract with the addon-contrib CI pipeline (verify,
#                  build, test-unit, test-integration, test-e2e, test-chart)
#   Development  - `make lint`, `make template`, `make status`
#   Help         - `make help`
#
# `_deploy-*` targets are composition helpers: they assume their inputs are
# already validated by a public deploy-* wrapper and are intentionally
# underscored so end users avoid calling them directly.
# -----------------------------------------------------------------------------

# -----------------------------------------------------------------------------
# Tool binaries
#
# Kept overrideable so the same recipes work on hosts that need `sudo docker`
# (no rootless setup, no docker group), inside CI where helm/kubectl come
# from a custom path, and during local dev where the user may want to point
# at kind/k3d-vended versions. CHAIN_DOCKER_COMPOSE below inspects the first
# word of $(DOCKER) to detect the sudo case, so callers MUST keep the
# `sudo docker` form (not `docker sudo`, not aliases).
# -----------------------------------------------------------------------------
KUBECTL ?= kubectl
HELM ?= helm
DOCKER ?= docker
BATS ?= bats

# -----------------------------------------------------------------------------
# OCM and chart identity
# -----------------------------------------------------------------------------
NAMESPACE ?= open-cluster-management
RELEASE_NAME ?= flock-addon
CHART_PATH ?= charts/flock-addon

# Test fixture locations. Both directories are checked in and discovered
# automatically by the test runners (helm-unittest scans
# `<chart>/tests/*_test.yaml`, bats picks up every *.bats below
# tests/entrypoint/).
TESTS_CHART_DIR ?= $(CHART_PATH)/tests
TESTS_ENTRYPOINT_DIR ?= tests/entrypoint

# -----------------------------------------------------------------------------
# Image coordinates
#
# FLOCK_ALLIANCE_IMAGE is the single source of truth for the workload image.
# When callers override it (e.g. with a privately mirrored repo), the deploy
# recipes split it back into image.repository / image.tag for Helm using
# `rev | cut -d: -f1`. We split at the LAST colon — using `cut -d: -f1`
# without `rev` would mis-split values like `registry.local:5000/foo:tag`,
# which is the exact reason rev/cut/rev is used everywhere image=... appears.
# -----------------------------------------------------------------------------
IMAGE_REGISTRY ?= ghcr.io
IMAGE_OWNER ?= flock-io
IMAGE_NAME ?= fl-alliance-client
IMAGE_TAG ?= latest
IMAGE_PULL_POLICY ?= Always
FLOCK_ALLIANCE_IMAGE ?= $(IMAGE_REGISTRY)/$(IMAGE_OWNER)/$(IMAGE_NAME):$(IMAGE_TAG)
IMAGE_PULL_SECRET ?=

# -----------------------------------------------------------------------------
# Local-chain mode prerequisites
#
# Required only by the hub-managed local-chain deploy recipes
# (deploy-local-chain-s3*). They intentionally have no defaults so the
# recipes can `$(error ...)` with an actionable message instead of silently
# pointing at a non-existent path or contacting the wrong RPC host.
# -----------------------------------------------------------------------------
FL_ALLIANCE_CLIENT_DIR ?=
MODEL_ARCHIVE ?=
MODEL_HASH ?=
RPC_HOST ?=

# -----------------------------------------------------------------------------
# Compose forwarding
#
# When DOCKER='sudo docker', `make chain` in FL-Alliance-Client must invoke
# `sudo -E docker compose ...` (note `-E`: the chain target reads variables
# like MODEL_DEFINITION_HASH from the environment, which sudo strips by
# default). When DOCKER='docker', plain `docker compose` is enough. We
# detect by checking whether the first word of $(DOCKER) is `sudo`.
# -----------------------------------------------------------------------------
CHAIN_DOCKER_COMPOSE ?= $(if $(filter sudo,$(word 1,$(DOCKER))),sudo -E $(wordlist 2,$(words $(DOCKER)),$(DOCKER)) compose,$(DOCKER) compose)

# -----------------------------------------------------------------------------
# Local MinIO defaults (nami storage backend)
#
# These back the hub-managed MinIO instance used by the
# deploy-local-chain-s3-compatible flow. The container is bound to
# host-network ports so managed clusters can reach it as
# http://<RPC_HOST>:$(S3_COMPAT_PORT).
#
# Credentials and bucket name are intentionally task-scoped:
#
#   * deploy-local-chain-s3-compatible generates a fresh access/secret pair
#     (openssl rand) per invocation and passes them down to MinIO + the
#     AddOnDeploymentConfig in lockstep. Every FL client on every managed
#     cluster therefore receives the SAME credentials via OCM customizedVariable
#     substitution — multi-client coordination is preserved exactly as before;
#     only the value being propagated changes from the well-known
#     `minioadmin/minioadmin` to a per-task random pair.
#
#   * The bucket name is derived from MODEL_HASH (`flock-task-<sha256[:12]>`),
#     so concurrent tasks on the same hub never clobber each other's artifacts
#     and `make undeploy-local-chain` can recycle a task's bucket without
#     touching others.
#
# Operator-facing overrides (use either to pin known values, e.g. for CI):
#   MINIO_ACCESS_KEY=... MINIO_SECRET_KEY=...  -> skip random generation
#   MINIO_BUCKET=...                            -> skip MODEL_HASH derivation
#
# The S3_COMPAT_* variables below are still honored when running
# `start-local-s3-compatible` standalone (e.g. for manual `mc cp` testing
# from an operator laptop) — those defaults intentionally stay on the
# legacy `minioadmin` values so quick smoke tests don't require a fresh
# generation step.
# -----------------------------------------------------------------------------
MINIO_IMAGE ?= quay.io/minio/minio
MINIO_MC_IMAGE ?= quay.io/minio/mc
S3_COMPAT_CONTAINER_NAME ?= flock-minio
S3_COMPAT_DATA_DIR ?= /srv/flock-minio/data
S3_COMPAT_PORT ?= 9000
S3_COMPAT_CONSOLE_PORT ?= 9001
S3_COMPAT_ACCESS_KEY ?= minioadmin
S3_COMPAT_SECRET_KEY ?= minioadmin
S3_COMPAT_BUCKET ?= flock-dev
S3_COMPAT_REGION ?= us-east-1
S3_COMPAT_ADDRESSING_STYLE ?= path
S3_COMPAT_VERIFY_SSL ?= false

# Operator overrides for `make deploy-local-chain-s3-compatible`. Empty
# defaults trigger random generation inside the recipe.
MINIO_ACCESS_KEY ?=
MINIO_SECRET_KEY ?=
MINIO_BUCKET ?=

# -----------------------------------------------------------------------------
# Conditional pull-secret injection
#
# Helm rejects an empty entry inside the pullSecrets list (it would render
# a Pod with `imagePullSecrets: [{name: ""}]`, which fails admission). Only
# emit the --set flag when the user actually supplied a Secret name; this
# keeps the default render valid for users on public registries.
# -----------------------------------------------------------------------------
ifneq ($(strip $(IMAGE_PULL_SECRET)),)
IMAGE_PULL_SECRET_ARG := --set image.pullSecrets[0]='$(IMAGE_PULL_SECRET)'
endif

.PHONY: all
all: help

##@ Deployment

# -----------------------------------------------------------------------------
# Public deploy targets
#
# Three public entrypoints with intentionally different authority models:
#
#   make deploy
#       Install the chart on the hub with default values.yaml settings
#       (Manual install strategy, storage.backend=s3, empty chain
#       addresses). Nodes supply TASK_ADDRESS / TOKEN_ADDRESS /
#       BLOCKCHAIN_RPC / PRIVATE_KEY via their per-node `.env`. This is
#       the canonical OCM flow most operators expect:
#           make deploy
#           make enable-addon CLUSTER=m1
#           make enable-addon CLUSTER=m2
#           ...
#
#   make deploy-testnet TASK_ADDRESS=0x... [TOKEN_ADDRESS=0x...]
#       Same layout but the hub centrally pins task (and optionally
#       token) contract addresses so `make update-task` can rotate them
#       without touching every node's `.env`. Use on FLock's public
#       testnet when operators want fleet-wide control of the task.
#
#   make deploy-local-chain-s3 / deploy-local-chain-s3-compatible
#       Hub bootstraps its own FlockToken/FlockTask on a local RPC (and,
#       in the -compatible variant, a local MinIO), then calls the
#       internal `_deploy-local-chain-*` recipes to install the chart
#       with every hub-side address already filled in. For unfunded
#       local-chain smoke tests and fully offline demos.
#
# `_deploy-*` targets are underscored because they assume their inputs
# are already validated by the outer deploy-local-chain-* flows; end
# users should never call the underscored variants directly.
# -----------------------------------------------------------------------------

# -----------------------------------------------------------------------------
# deploy
#
# The bare minimum: install the chart with defaults. Chain addresses and
# PRIVATE_KEY come from each managed cluster's `.env`; the entrypoint
# script's hub-vs-.env precedence rule means operators can later run any
# `deploy-testnet` or `update-task` variant to take hub-side control
# without re-enabling the addon per cluster. Storage backend stays on the
# values.yaml default (`s3`) so FLock's testnet signer-authenticated
# uploads work out of the box.
# -----------------------------------------------------------------------------
.PHONY: deploy
deploy: ## Install the addon control plane on the hub with default values (enable per cluster with `make enable-addon`)
	$(HELM) upgrade --install $(RELEASE_NAME) $(CHART_PATH) \
		--set image.repository=$$(echo $(FLOCK_ALLIANCE_IMAGE) | rev | cut -d: -f2- | rev) \
		--set image.tag=$$(echo $(FLOCK_ALLIANCE_IMAGE) | rev | cut -d: -f1 | rev) \
		--set image.pullPolicy='$(IMAGE_PULL_POLICY)' \
		$(IMAGE_PULL_SECRET_ARG)

.PHONY: deploy-testnet
deploy-testnet: ## Deploy addon for testnet mode (usage: make deploy-testnet TASK_ADDRESS=0x... [TOKEN_ADDRESS=0x...])
ifndef TASK_ADDRESS
	$(error TASK_ADDRESS is not set. Usage: make deploy-testnet TASK_ADDRESS=0x... [TOKEN_ADDRESS=0x...])
endif
	$(HELM) upgrade --install $(RELEASE_NAME) $(CHART_PATH) \
		--set addon.installStrategy=Manual \
		--set placement.gpu.enabled=false \
		--set placement.all.enabled=false \
		--set image.repository=$$(echo $(FLOCK_ALLIANCE_IMAGE) | rev | cut -d: -f2- | rev) \
		--set image.tag=$$(echo $(FLOCK_ALLIANCE_IMAGE) | rev | cut -d: -f1 | rev) \
		--set image.pullPolicy='$(IMAGE_PULL_POLICY)' \
		$(IMAGE_PULL_SECRET_ARG) \
		--set deploymentConfig.blockchain.tokenAddress='$(TOKEN_ADDRESS)' \
		--set deploymentConfig.blockchain.taskAddress='$(TASK_ADDRESS)' \
		--set deploymentConfig.storage.backend='s3'

.PHONY: _deploy-local-chain-s3-compatible
_deploy-local-chain-s3-compatible:
ifndef RPC
	$(error RPC is not set. Usage: make _deploy-local-chain-s3-compatible RPC=http://<hub-ip>:8545 [TOKEN_ADDRESS=0x...] [TASK_ADDRESS=0x...])
endif
	$(HELM) upgrade --install $(RELEASE_NAME) $(CHART_PATH) \
		--set addon.installStrategy=Manual \
		--set placement.gpu.enabled=false \
		--set placement.all.enabled=false \
		--set image.repository=$$(echo $(FLOCK_ALLIANCE_IMAGE) | rev | cut -d: -f2- | rev) \
		--set image.tag=$$(echo $(FLOCK_ALLIANCE_IMAGE) | rev | cut -d: -f1 | rev) \
		--set image.pullPolicy='$(IMAGE_PULL_POLICY)' \
		$(IMAGE_PULL_SECRET_ARG) \
		--set deploymentConfig.blockchain.rpc='$(RPC)' \
		--set deploymentConfig.blockchain.tokenAddress='$(TOKEN_ADDRESS)' \
		--set deploymentConfig.blockchain.taskAddress='$(TASK_ADDRESS)' \
		--set deploymentConfig.storage.s3Compat.endpointUrl='$(S3_COMPAT_ENDPOINT_URL)' \
		--set deploymentConfig.storage.s3Compat.bucket='$(S3_COMPAT_BUCKET)' \
		--set deploymentConfig.storage.s3Compat.accessKey='$(S3_COMPAT_ACCESS_KEY)' \
		--set deploymentConfig.storage.s3Compat.secretKey='$(S3_COMPAT_SECRET_KEY)' \
		--set deploymentConfig.storage.s3Compat.region='$(S3_COMPAT_REGION)' \
		--set deploymentConfig.storage.s3Compat.addressingStyle='$(S3_COMPAT_ADDRESSING_STYLE)' \
		--set deploymentConfig.storage.s3Compat.verifySsl='$(S3_COMPAT_VERIFY_SSL)' \
		--set deploymentConfig.storage.backend='nami'

.PHONY: _deploy-local-chain-s3
_deploy-local-chain-s3:
ifndef RPC
	$(error RPC is not set. Usage: make _deploy-local-chain-s3 RPC=http://<hub-ip>:8545 [TOKEN_ADDRESS=0x...] [TASK_ADDRESS=0x...])
endif
	$(HELM) upgrade --install $(RELEASE_NAME) $(CHART_PATH) \
		--set addon.installStrategy=Manual \
		--set placement.gpu.enabled=false \
		--set placement.all.enabled=false \
		--set image.repository=$$(echo $(FLOCK_ALLIANCE_IMAGE) | rev | cut -d: -f2- | rev) \
		--set image.tag=$$(echo $(FLOCK_ALLIANCE_IMAGE) | rev | cut -d: -f1 | rev) \
		--set image.pullPolicy='$(IMAGE_PULL_POLICY)' \
		$(IMAGE_PULL_SECRET_ARG) \
		--set deploymentConfig.blockchain.rpc='$(RPC)' \
		--set deploymentConfig.blockchain.tokenAddress='$(TOKEN_ADDRESS)' \
		--set deploymentConfig.blockchain.taskAddress='$(TASK_ADDRESS)' \
		--set deploymentConfig.storage.backend='s3'

# -----------------------------------------------------------------------------
# Local MinIO lifecycle
#
# The readiness loop below uses `mc ready` (not a fixed sleep) because
# MinIO cold-start time varies wildly between SSD and rotational backing
# stores. `mc ready` polls /minio/health/ready, which only succeeds once
# MinIO has finished initialising disks and elected its quorum — a
# stronger guarantee than /minio/health/live. The 30-second cap is
# generous enough for the slowest disks we have observed in CI but still
# fails fast on misconfig.
# -----------------------------------------------------------------------------

.PHONY: start-local-s3-compatible
start-local-s3-compatible: ## Start a local MinIO instance on the hub and ensure the bucket exists
	@set -euo pipefail; \
	mkdir -p '$(S3_COMPAT_DATA_DIR)'; \
	echo "[flock-addon] ensuring local MinIO container: $(S3_COMPAT_CONTAINER_NAME)"; \
	if $(DOCKER) ps -a --format '{{.Names}}' | grep -qx '$(S3_COMPAT_CONTAINER_NAME)'; then \
		echo "[flock-addon] removing pre-existing container $(S3_COMPAT_CONTAINER_NAME)"; \
		$(DOCKER) rm -f '$(S3_COMPAT_CONTAINER_NAME)' >/dev/null; \
	fi; \
	$(DOCKER) run -d \
		--name '$(S3_COMPAT_CONTAINER_NAME)' \
		-p '$(S3_COMPAT_PORT):9000' \
		-p '$(S3_COMPAT_CONSOLE_PORT):9001' \
		-e MINIO_ROOT_USER='$(S3_COMPAT_ACCESS_KEY)' \
		-e MINIO_ROOT_PASSWORD='$(S3_COMPAT_SECRET_KEY)' \
		-v '$(S3_COMPAT_DATA_DIR):/data' \
		'$(MINIO_IMAGE)' server /data --console-address ":9001" >/dev/null; \
	echo "[flock-addon] MinIO listening at http://127.0.0.1:$(S3_COMPAT_PORT) (console http://127.0.0.1:$(S3_COMPAT_CONSOLE_PORT))"; \
	for i in $$(seq 1 30); do \
		if $(DOCKER) run --rm --network host --entrypoint /bin/sh '$(MINIO_MC_IMAGE)' -ec \
			'mc alias set local http://127.0.0.1:$(S3_COMPAT_PORT) "$(S3_COMPAT_ACCESS_KEY)" "$(S3_COMPAT_SECRET_KEY)" >/dev/null && mc ready local >/dev/null 2>&1'; then \
			echo "[flock-addon] MinIO ready after $$i second(s)"; \
			break; \
		fi; \
		if [ $$i -eq 30 ]; then \
			echo "[flock-addon] ERROR: MinIO did not become ready within 30s" >&2; \
			$(DOCKER) logs --tail=40 '$(S3_COMPAT_CONTAINER_NAME)' >&2 || true; \
			exit 1; \
		fi; \
		sleep 1; \
	done; \
	$(DOCKER) run --rm --network host --entrypoint /bin/sh '$(MINIO_MC_IMAGE)' -ec '\
		mc alias set local http://127.0.0.1:$(S3_COMPAT_PORT) "$(S3_COMPAT_ACCESS_KEY)" "$(S3_COMPAT_SECRET_KEY)" >/dev/null && \
		mc mb --ignore-existing local/$(S3_COMPAT_BUCKET) >/dev/null'; \
	echo "[flock-addon] bucket ensured: $(S3_COMPAT_BUCKET)"

.PHONY: stop-local-s3-compatible
stop-local-s3-compatible: ## Stop and remove the local MinIO container (leaves data dir intact)
	@set -eu; \
	if $(DOCKER) ps -a --format '{{.Names}}' | grep -qx '$(S3_COMPAT_CONTAINER_NAME)'; then \
		echo "[flock-addon] removing container $(S3_COMPAT_CONTAINER_NAME)"; \
		$(DOCKER) rm -f '$(S3_COMPAT_CONTAINER_NAME)' >/dev/null; \
	else \
		echo "[flock-addon] no container named $(S3_COMPAT_CONTAINER_NAME); nothing to do"; \
	fi

# -----------------------------------------------------------------------------
# Hub-managed local-chain flows
#
# `make chain` in FL-Alliance-Client deploys FlockToken/FlockTask and writes
# their addresses to data/contracts.json. We parse that JSON with python3
# (already required as a runtime dependency of FL-Alliance-Client) instead
# of jq to avoid pulling another tool onto the hub host.
# -----------------------------------------------------------------------------

.PHONY: deploy-local-chain-s3
deploy-local-chain-s3: ## Auto-start local chain on the Hub using a pre-uploaded MODEL_HASH, then deploy the addon with the original signer-based S3 storage
ifndef FL_ALLIANCE_CLIENT_DIR
	$(error FL_ALLIANCE_CLIENT_DIR is not set. Usage: make deploy-local-chain-s3 FL_ALLIANCE_CLIENT_DIR=/path/to/FL-Alliance-Client MODEL_HASH=<sha256> RPC_HOST=<hub-ip>)
endif
ifndef MODEL_HASH
	$(error MODEL_HASH is not set. Usage: make deploy-local-chain-s3 FL_ALLIANCE_CLIENT_DIR=/path/to/FL-Alliance-Client MODEL_HASH=<sha256> RPC_HOST=<hub-ip>)
endif
ifndef RPC_HOST
	$(error RPC_HOST is not set. Usage: make deploy-local-chain-s3 FL_ALLIANCE_CLIENT_DIR=/path/to/FL-Alliance-Client MODEL_HASH=<sha256> RPC_HOST=<hub-ip>)
endif
	@set -euo pipefail; \
	echo "[flock-addon] MODEL_HASH=$(MODEL_HASH)"; \
	$(MAKE) -C '$(FL_ALLIANCE_CLIENT_DIR)' chain \
		$(if $(strip $(CHAIN_DOCKER_COMPOSE)),DOCKER_COMPOSE='$(CHAIN_DOCKER_COMPOSE)') \
		MODEL_DEFINITION_HASH='$(MODEL_HASH)' \
		$(if $(strip $(ROUNDS)),ROUNDS='$(ROUNDS)') \
		$(if $(strip $(MIN_PARTICIPANTS)),MIN_PARTICIPANTS='$(MIN_PARTICIPANTS)') \
		$(if $(strip $(TIME_TO_PROPOSE)),TIME_TO_PROPOSE='$(TIME_TO_PROPOSE)') \
		$(if $(strip $(TIME_TO_VOTE)),TIME_TO_VOTE='$(TIME_TO_VOTE)'); \
	CONTRACTS='$(FL_ALLIANCE_CLIENT_DIR)/data/contracts.json'; \
	if [ ! -f "$$CONTRACTS" ]; then \
		echo "[flock-addon] ERROR: $$CONTRACTS not found; did 'make chain' fail?" >&2; \
		exit 1; \
	fi; \
	TOKEN_ADDRESS=$$(python3 -c "import json,sys; print(json.load(open(sys.argv[1]))['FlockToken']['address'])" "$$CONTRACTS"); \
	TASK_ADDRESS=$$(python3 -c "import json,sys; print(json.load(open(sys.argv[1]))['FlockTask']['address'])" "$$CONTRACTS"); \
	if [ -z "$$TOKEN_ADDRESS" ] || [ -z "$$TASK_ADDRESS" ]; then \
		echo "[flock-addon] ERROR: FL-Alliance-Client chain deployment did not produce TOKEN_ADDRESS/TASK_ADDRESS" >&2; \
		exit 1; \
	fi; \
	echo "[flock-addon] TOKEN_ADDRESS=$$TOKEN_ADDRESS"; \
	echo "[flock-addon] TASK_ADDRESS=$$TASK_ADDRESS"; \
	$(MAKE) _deploy-local-chain-s3 \
		RPC='http://$(RPC_HOST):8545' \
		TOKEN_ADDRESS="$$TOKEN_ADDRESS" \
		TASK_ADDRESS="$$TASK_ADDRESS"; \
	echo ""; \
	echo "============================================================"; \
	echo "[flock-addon] local-chain + signer-S3 deployment complete"; \
	echo "  TASK_ADDRESS  = $$TASK_ADDRESS"; \
	echo "  TOKEN_ADDRESS = $$TOKEN_ADDRESS"; \
	echo "  CHAIN_RPC     = http://$(RPC_HOST):8545"; \
	echo "  STORAGE       = signer-authenticated S3 (FLock testnet)"; \
	echo "  MODEL_HASH    = $(MODEL_HASH)"; \
	echo "============================================================"; \
	echo "Next steps:"; \
	echo "  1. Enable the addon on each managed cluster:"; \
	echo "       make enable-addon CLUSTER=<name>"; \
	echo "  2. Tear everything down when the task is finished:"; \
	echo "       make undeploy-local-chain FL_ALLIANCE_CLIENT_DIR='$(FL_ALLIANCE_CLIENT_DIR)'"; \
	echo "============================================================"

# -----------------------------------------------------------------------------
# deploy-local-chain-s3-compatible
#
# The model's sha256 doubles as the on-chain MODEL_DEFINITION_HASH and as
# the S3 object key. Computing it once locally keeps the upload step and
# the `make chain` step in agreement on the same identifier without an
# extra RPC round-trip. The model directory is bind-mounted read-only so
# `mc cp` cannot accidentally mutate the operator's source artefacts.
# -----------------------------------------------------------------------------

.PHONY: deploy-local-chain-s3-compatible
deploy-local-chain-s3-compatible: ## Auto-start local chain + local S3-compatible storage on the Hub, upload MODEL_ARCHIVE, then deploy the addon
ifndef FL_ALLIANCE_CLIENT_DIR
	$(error FL_ALLIANCE_CLIENT_DIR is not set. Usage: make deploy-local-chain-s3-compatible FL_ALLIANCE_CLIENT_DIR=/path/to/FL-Alliance-Client MODEL_ARCHIVE=/path/to/model.tar.gz RPC_HOST=<hub-ip>)
endif
ifndef MODEL_ARCHIVE
	$(error MODEL_ARCHIVE is not set. Usage: make deploy-local-chain-s3-compatible FL_ALLIANCE_CLIENT_DIR=/path/to/FL-Alliance-Client MODEL_ARCHIVE=/path/to/model.tar.gz RPC_HOST=<hub-ip>)
endif
ifndef RPC_HOST
	$(error RPC_HOST is not set. Usage: make deploy-local-chain-s3-compatible FL_ALLIANCE_CLIENT_DIR=/path/to/FL-Alliance-Client MODEL_ARCHIVE=/path/to/model.tar.gz RPC_HOST=<hub-ip>)
endif
	@set -euo pipefail; \
	if [ ! -f '$(MODEL_ARCHIVE)' ]; then \
		echo "[flock-addon] ERROR: MODEL_ARCHIVE does not exist: $(MODEL_ARCHIVE)" >&2; \
		exit 1; \
	fi; \
	if ! command -v openssl >/dev/null 2>&1; then \
		echo "[flock-addon] ERROR: openssl is required to generate per-task MinIO credentials" >&2; \
		exit 1; \
	fi; \
	MODEL_HASH=$$(sha256sum '$(MODEL_ARCHIVE)' | cut -d' ' -f1); \
	MODEL_DIR=$$(cd "$$(dirname '$(MODEL_ARCHIVE)')" && pwd); \
	MODEL_BASENAME=$$(basename '$(MODEL_ARCHIVE)'); \
	S3_ENDPOINT='http://$(RPC_HOST):$(S3_COMPAT_PORT)'; \
	ACCESS_KEY='$(MINIO_ACCESS_KEY)'; [ -n "$$ACCESS_KEY" ] || ACCESS_KEY=$$(openssl rand -hex 12); \
	SECRET_KEY='$(MINIO_SECRET_KEY)'; [ -n "$$SECRET_KEY" ] || SECRET_KEY=$$(openssl rand -hex 24); \
	BUCKET='$(MINIO_BUCKET)'; [ -n "$$BUCKET" ] || BUCKET=flock-task-$$(echo "$$MODEL_HASH" | cut -c1-12); \
	echo "[flock-addon] MODEL_HASH=$$MODEL_HASH"; \
	echo "[flock-addon] task-scoped bucket: $$BUCKET"; \
	echo "[flock-addon] task-scoped MinIO credentials: ACCESS_KEY=$$ACCESS_KEY (SECRET_KEY hidden; use 'make minio-credentials' to fetch)"; \
	$(MAKE) start-local-s3-compatible \
		S3_COMPAT_ACCESS_KEY="$$ACCESS_KEY" \
		S3_COMPAT_SECRET_KEY="$$SECRET_KEY" \
		S3_COMPAT_BUCKET="$$BUCKET"; \
	echo "[flock-addon] uploading $(MODEL_ARCHIVE) to bucket $$BUCKET with key $$MODEL_HASH"; \
	$(DOCKER) run --rm --network host --entrypoint /bin/sh \
		-e MODEL_BASENAME="$$MODEL_BASENAME" \
		-e MODEL_HASH="$$MODEL_HASH" \
		-e BUCKET="$$BUCKET" \
		-e ACCESS_KEY="$$ACCESS_KEY" \
		-e SECRET_KEY="$$SECRET_KEY" \
		-v "$$MODEL_DIR:/upload:ro" '$(MINIO_MC_IMAGE)' -ec '\
		mc alias set local http://127.0.0.1:$(S3_COMPAT_PORT) "$$ACCESS_KEY" "$$SECRET_KEY" >/dev/null && \
		mc cp "/upload/$${MODEL_BASENAME}" "local/$${BUCKET}/$${MODEL_HASH}" >/dev/null'; \
	$(MAKE) -C '$(FL_ALLIANCE_CLIENT_DIR)' chain \
		$(if $(strip $(CHAIN_DOCKER_COMPOSE)),DOCKER_COMPOSE='$(CHAIN_DOCKER_COMPOSE)') \
		MODEL_DEFINITION_HASH="$$MODEL_HASH" \
		$(if $(strip $(ROUNDS)),ROUNDS='$(ROUNDS)') \
		$(if $(strip $(MIN_PARTICIPANTS)),MIN_PARTICIPANTS='$(MIN_PARTICIPANTS)') \
		$(if $(strip $(TIME_TO_PROPOSE)),TIME_TO_PROPOSE='$(TIME_TO_PROPOSE)') \
		$(if $(strip $(TIME_TO_VOTE)),TIME_TO_VOTE='$(TIME_TO_VOTE)'); \
	CONTRACTS='$(FL_ALLIANCE_CLIENT_DIR)/data/contracts.json'; \
	if [ ! -f "$$CONTRACTS" ]; then \
		echo "[flock-addon] ERROR: $$CONTRACTS not found; did 'make chain' fail?" >&2; \
		exit 1; \
	fi; \
	TOKEN_ADDRESS=$$(python3 -c "import json,sys; print(json.load(open(sys.argv[1]))['FlockToken']['address'])" "$$CONTRACTS"); \
	TASK_ADDRESS=$$(python3 -c "import json,sys; print(json.load(open(sys.argv[1]))['FlockTask']['address'])" "$$CONTRACTS"); \
	if [ -z "$$TOKEN_ADDRESS" ] || [ -z "$$TASK_ADDRESS" ]; then \
		echo "[flock-addon] ERROR: FL-Alliance-Client chain deployment did not produce TOKEN_ADDRESS/TASK_ADDRESS" >&2; \
		exit 1; \
	fi; \
	echo "[flock-addon] TOKEN_ADDRESS=$$TOKEN_ADDRESS"; \
	echo "[flock-addon] TASK_ADDRESS=$$TASK_ADDRESS"; \
	$(MAKE) _deploy-local-chain-s3-compatible \
		RPC='http://$(RPC_HOST):8545' \
		TOKEN_ADDRESS="$$TOKEN_ADDRESS" \
		TASK_ADDRESS="$$TASK_ADDRESS" \
		S3_COMPAT_ENDPOINT_URL="$$S3_ENDPOINT" \
		S3_COMPAT_BUCKET="$$BUCKET" \
		S3_COMPAT_ACCESS_KEY="$$ACCESS_KEY" \
		S3_COMPAT_SECRET_KEY="$$SECRET_KEY" \
		S3_COMPAT_REGION='$(S3_COMPAT_REGION)' \
		S3_COMPAT_ADDRESSING_STYLE='$(S3_COMPAT_ADDRESSING_STYLE)' \
		S3_COMPAT_VERIFY_SSL='$(S3_COMPAT_VERIFY_SSL)'; \
	echo ""; \
	echo "============================================================"; \
	echo "[flock-addon] local-chain + S3-compatible deployment complete"; \
	echo "  TASK_ADDRESS  = $$TASK_ADDRESS"; \
	echo "  TOKEN_ADDRESS = $$TOKEN_ADDRESS"; \
	echo "  CHAIN_RPC     = http://$(RPC_HOST):8545"; \
	echo "  S3_ENDPOINT   = $$S3_ENDPOINT"; \
	echo "  S3_BUCKET     = $$BUCKET"; \
	echo "  ACCESS_KEY    = $$ACCESS_KEY"; \
	echo "  SECRET_KEY    = (hidden — run 'make minio-credentials')"; \
	echo "  MODEL_KEY     = $$BUCKET/$$MODEL_HASH"; \
	echo "============================================================"; \
	echo "Next steps:"; \
	echo "  1. Enable the addon on each managed cluster:"; \
	echo "       make enable-addon CLUSTER=<name>"; \
	echo "  2. Tear everything down when the task is finished:"; \
	echo "       make undeploy-local-chain FL_ALLIANCE_CLIENT_DIR='$(FL_ALLIANCE_CLIENT_DIR)'"; \
	echo "============================================================"

# -----------------------------------------------------------------------------
# Lifecycle helpers
#
# `--reuse-values` keeps every other field at its current rendered value so
# operators can rotate task addresses without re-supplying RPC, image, or
# storage settings. Each managed cluster picks the new value up on its next
# AddOnDeploymentConfig reconcile (see docs/troubleshooting.md for
# force-rollout). OCM's addon-manager owns the per-cluster Namespace and
# Deployment objects via the AddOnTemplate; deleting the hub-side
# ClusterManagementAddOn is what triggers GC on the managed clusters, so
# `helm uninstall` is the correct (and only) way to fully remove the addon.
# -----------------------------------------------------------------------------

.PHONY: update-task
update-task: ## Update the shared task address only (usage: make update-task TASK_ADDRESS=0x...)
ifndef TASK_ADDRESS
	$(error TASK_ADDRESS is not set. Usage: make update-task TASK_ADDRESS=0x...)
endif
	$(HELM) upgrade $(RELEASE_NAME) $(CHART_PATH) \
		--reuse-values \
		--set deploymentConfig.blockchain.taskAddress='$(TASK_ADDRESS)'

.PHONY: undeploy
undeploy: ## Remove the FLock addon Helm release (managed cluster workloads are garbage-collected by OCM)
	@echo "[flock-addon] uninstalling Helm release $(RELEASE_NAME)"
	$(HELM) uninstall $(RELEASE_NAME) || true

.PHONY: undeploy-all
undeploy-all: undeploy stop-local-s3-compatible ## Full local cleanup: helm uninstall + stop local MinIO container

# -----------------------------------------------------------------------------
# Local-chain task helpers
#
# `make deploy-local-chain-s3-compatible` materialises three task-scoped
# pieces of state that the chart's defaults do not encode: the MinIO
# bucket, the access/secret pair, and the FlockToken/FlockTask addresses.
# All three live inside the chart's AddOnDeploymentConfig as plain
# customizedVariable entries because OCM substitutes them into the FL
# client Pod env on every managed cluster — that's also what guarantees
# every cluster sees the same values, which is the foundation of
# multi-client coordination.
#
# `minio-credentials` reads them back out via kubectl jsonpath so an
# operator can drive `mc` from their laptop without re-running the deploy.
#
# `undeploy-local-chain` is the reverse of `deploy-local-chain-*`:
#   1. Read the task's bucket + creds from the live ADC (best effort —
#      may be empty if the chart was never installed).
#   2. Drop the bucket via `mc rb --force` so we don't leak per-task data
#      into /srv/flock-minio when the next task reuses the hub.
#   3. helm uninstall — OCM addon-manager garbage-collects the per-cluster
#      ManifestWork and the FL client Deployments cascade away.
#   4. Stop the MinIO container (idempotent; matches the ephemeral model).
#   5. Optionally stop the chain when FL_ALLIANCE_CLIENT_DIR is provided —
#      we call `make chain-stop` defensively (|| true) because the upstream
#      target name has changed historically and the operator shouldn't have
#      a half-failed undeploy block them from creating the next task.
# -----------------------------------------------------------------------------

.PHONY: minio-credentials
minio-credentials: ## Print the current task's MinIO credentials, bucket, and endpoint URL (read from the live AddOnDeploymentConfig)
	@set -eu; \
	if ! $(KUBECTL) -n $(NAMESPACE) get addondeploymentconfig $(RELEASE_NAME)-config >/dev/null 2>&1; then \
		echo "[flock-addon] ERROR: AddOnDeploymentConfig $(RELEASE_NAME)-config not found in namespace $(NAMESPACE)." >&2; \
		echo "[flock-addon] Hint: run 'make deploy-local-chain-s3-compatible ...' first." >&2; \
		exit 1; \
	fi; \
	get_var() { \
		$(KUBECTL) -n $(NAMESPACE) get addondeploymentconfig $(RELEASE_NAME)-config \
			-o jsonpath="{range .spec.customizedVariables[?(@.name==\"$$1\")]}{.value}{end}"; \
	}; \
	BACKEND=$$(get_var STORAGE_BACKEND); \
	BUCKET=$$(get_var S3_COMPAT_BUCKET); \
	if [ "$$BACKEND" != "nami" ] || [ -z "$$BUCKET" ]; then \
		echo "[flock-addon] No active MinIO-backed task found (STORAGE_BACKEND=$$BACKEND, BUCKET=$$BUCKET)."; \
		echo "[flock-addon] The local-chain MinIO flow is only used by 'make deploy-local-chain-s3-compatible'."; \
		exit 0; \
	fi; \
	ENDPOINT=$$(get_var S3_COMPAT_ENDPOINT_URL); \
	ACCESS_KEY=$$(get_var S3_COMPAT_ACCESS_KEY); \
	SECRET_KEY=$$(get_var S3_COMPAT_SECRET_KEY); \
	echo "ENDPOINT_URL=$$ENDPOINT"; \
	echo "BUCKET=$$BUCKET"; \
	echo "ACCESS_KEY=$$ACCESS_KEY"; \
	echo "SECRET_KEY=$$SECRET_KEY"; \
	echo ""; \
	echo "# Explore the bucket from this hub:"; \
	echo "mc alias set hubminio $$ENDPOINT $$ACCESS_KEY $$SECRET_KEY"; \
	echo "mc ls hubminio/$$BUCKET/"

.PHONY: undeploy-local-chain
undeploy-local-chain: ## Tear down a local-chain task: drop bucket + helm uninstall + stop MinIO + optional chain stop (pass FL_ALLIANCE_CLIENT_DIR=...)
	@set -eu; \
	if $(KUBECTL) -n $(NAMESPACE) get addondeploymentconfig $(RELEASE_NAME)-config >/dev/null 2>&1; then \
		get_var() { \
			$(KUBECTL) -n $(NAMESPACE) get addondeploymentconfig $(RELEASE_NAME)-config \
				-o jsonpath="{range .spec.customizedVariables[?(@.name==\"$$1\")]}{.value}{end}"; \
		}; \
		BACKEND=$$(get_var STORAGE_BACKEND); \
		BUCKET=$$(get_var S3_COMPAT_BUCKET); \
		ACCESS_KEY=$$(get_var S3_COMPAT_ACCESS_KEY); \
		SECRET_KEY=$$(get_var S3_COMPAT_SECRET_KEY); \
		if [ "$$BACKEND" = "nami" ] && [ -n "$$BUCKET" ] && [ -n "$$ACCESS_KEY" ] && [ -n "$$SECRET_KEY" ]; then \
			if $(DOCKER) ps --format '{{.Names}}' | grep -qx '$(S3_COMPAT_CONTAINER_NAME)'; then \
				echo "[flock-addon] removing task bucket: $$BUCKET"; \
				$(DOCKER) run --rm --network host --entrypoint /bin/sh \
					-e BUCKET="$$BUCKET" -e ACCESS_KEY="$$ACCESS_KEY" -e SECRET_KEY="$$SECRET_KEY" \
					'$(MINIO_MC_IMAGE)' -ec '\
					mc alias set local http://127.0.0.1:$(S3_COMPAT_PORT) "$$ACCESS_KEY" "$$SECRET_KEY" >/dev/null && \
					mc rb --force --dangerous "local/$$BUCKET" >/dev/null 2>&1' \
					|| echo "[flock-addon] WARN: bucket cleanup failed (already gone or mc error); continuing"; \
			else \
				echo "[flock-addon] MinIO container is not running; skipping bucket cleanup"; \
			fi; \
		else \
			echo "[flock-addon] no MinIO task scope detected in ADC; skipping bucket cleanup"; \
		fi; \
	else \
		echo "[flock-addon] AddOnDeploymentConfig not found; skipping bucket cleanup"; \
	fi; \
	echo "[flock-addon] uninstalling Helm release $(RELEASE_NAME)"; \
	$(HELM) uninstall $(RELEASE_NAME) || true; \
	$(MAKE) stop-local-s3-compatible; \
	if [ -n "$(FL_ALLIANCE_CLIENT_DIR)" ]; then \
		if [ -d "$(FL_ALLIANCE_CLIENT_DIR)" ]; then \
			echo "[flock-addon] attempting to stop chain via $(FL_ALLIANCE_CLIENT_DIR) (best effort)"; \
			$(MAKE) -C '$(FL_ALLIANCE_CLIENT_DIR)' chain-stop \
				$(if $(strip $(CHAIN_DOCKER_COMPOSE)),DOCKER_COMPOSE='$(CHAIN_DOCKER_COMPOSE)') 2>/dev/null \
				|| echo "[flock-addon] WARN: 'chain-stop' target not available in $(FL_ALLIANCE_CLIENT_DIR); stop the chain manually."; \
		else \
			echo "[flock-addon] WARN: FL_ALLIANCE_CLIENT_DIR=$(FL_ALLIANCE_CLIENT_DIR) does not exist; skipping chain stop."; \
		fi; \
	else \
		echo "[flock-addon] FL_ALLIANCE_CLIENT_DIR not set; leaving the chain running. Pass it to also stop the chain."; \
	fi; \
	echo "[flock-addon] local-chain teardown complete"

# -----------------------------------------------------------------------------
# enable-addon
#
# Builds the ManagedClusterAddOn manifest with `printf '%s\n'` rather than
# a heredoc. Heredocs in Make recipes are fragile because each
# backslash-newline turns the recipe into a single shell line: the heredoc
# terminator ends up on the same physical line as the body and the shell
# tries to interpret subsequent words as filenames. `printf` with one line
# per argument keeps every output line independent and pipes cleanly to
# `kubectl apply -f -`. The GPU label dispatch tolerates missing labels so
# unlabelled clusters fall through to the CPU variant.
# -----------------------------------------------------------------------------

.PHONY: enable-addon
enable-addon: ## Enable the addon on a managed cluster; auto-selects GPU/CPU template via managedcluster label `gpu=true` (override with CONFIG=/TEMPLATE=)
ifndef CLUSTER
	$(error CLUSTER is not set. Usage: make enable-addon CLUSTER=cluster1 [CONFIG=flock-addon-config] [TEMPLATE=flock-addon])
endif
	@set -eu; \
	GPU_LABEL=$$($(KUBECTL) get managedcluster $(CLUSTER) -o jsonpath='{.metadata.labels.gpu}' 2>/dev/null || true); \
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
	echo "[flock-addon] enabling addon on cluster=$(CLUSTER) template=$$TEMPLATE_NAME config=$$CONFIG_NAME"; \
	printf '%s\n' \
		'apiVersion: addon.open-cluster-management.io/v1alpha1' \
		'kind: ManagedClusterAddOn' \
		'metadata:' \
		'  name: $(RELEASE_NAME)' \
		'  namespace: $(CLUSTER)' \
		'  annotations:' \
		'    addon.open-cluster-management.io/v1alpha1-install-namespace: flock-system' \
		'spec:' \
		'  configs:' \
		'    - group: addon.open-cluster-management.io' \
		'      resource: addontemplates' \
		"      name: $$TEMPLATE_NAME" \
		'    - group: addon.open-cluster-management.io' \
		'      resource: addondeploymentconfigs' \
		"      name: $$CONFIG_NAME" \
		'      namespace: $(NAMESPACE)' \
		| $(KUBECTL) apply -f -

.PHONY: disable-addon
disable-addon: ## Disable the addon on a managed cluster (usage: make disable-addon CLUSTER=cluster1)
ifndef CLUSTER
	$(error CLUSTER is not set. Usage: make disable-addon CLUSTER=cluster1)
endif
	@echo "[flock-addon] disabling addon on cluster=$(CLUSTER)"
	$(KUBECTL) delete managedclusteraddon $(RELEASE_NAME) -n $(CLUSTER) --ignore-not-found

.PHONY: status
status: ## Print addon-related objects (hub-side)
	$(KUBECTL) get clustermanagementaddons
	$(KUBECTL) get addontemplates
	$(KUBECTL) get addondeploymentconfigs -A
	$(KUBECTL) get managedclusteraddons -A

##@ CI Targets

# -----------------------------------------------------------------------------
# CI contract
#
# The targets in this section satisfy the contract that addon-contrib's CI
# pipeline expects from every addon under this repo: verify, vendor, build,
# test-unit, test-integration, test-e2e, test-chart, image, image-push,
# image-manifest. Most are no-ops for this addon because we ship a Helm
# chart only — the actual container image is built and published from the
# FL-Alliance-Client repository — but the targets MUST exist or CI fails.
# -----------------------------------------------------------------------------

.PHONY: verify
verify: lint template test-unit test-integration ## Run the full offline gate: lint, template render, chart unit tests, entrypoint shell tests
	@echo "[flock-addon] verify complete"

.PHONY: vendor
vendor: ## No-op vendor target (Helm-only addon)
	@echo "[flock-addon] no vendor step required"

.PHONY: build
build: ## No-op build target (Helm-only addon; images are built in source repositories)
	@echo "[flock-addon] no build required"

# -----------------------------------------------------------------------------
# test-unit / test-integration
#
# Chart unit tests run helm-unittest against the rendered manifests so any
# change to a template, value, or fail-guard is caught before deployment.
# Integration tests exercise the embedded entrypoint script with bats,
# extracting it from a fresh `helm template`, replacing `exec` with a
# printf so argv can be captured, and asserting hub-vs-.env precedence,
# storage-backend guards, and the python argv shape — none of which need
# a real cluster. Both targets probe for their runner first because
# addon-contrib CI installs them on demand; printing the install command
# keeps local-dev parity with CI.
# -----------------------------------------------------------------------------

.PHONY: test-unit
test-unit: ## Chart unit tests via helm-unittest (covers all template branches and fail-guards)
	@if ! $(HELM) plugin list 2>/dev/null | awk '{print $$1}' | grep -qx unittest; then \
		echo "[flock-addon] helm-unittest plugin missing; install with:" >&2; \
		echo "    $(HELM) plugin install https://github.com/helm-unittest/helm-unittest" >&2; \
		exit 1; \
	fi
	$(HELM) unittest $(CHART_PATH)

.PHONY: test-integration
test-integration: ## Entrypoint shell tests via bats (hub-vs-.env precedence, storage guards, argv shape)
	@command -v $(BATS) >/dev/null 2>&1 || { \
		echo "[flock-addon] bats not found; install with:" >&2; \
		echo "    macOS:  brew install bats-core" >&2; \
		echo "    Linux:  apt-get install bats   (or build from https://github.com/bats-core/bats-core)" >&2; \
		exit 1; \
	}
	@HELM=$(HELM) $(BATS) $(TESTS_ENTRYPOINT_DIR)

# e2e tests require a live OCM hub and at least one managed cluster, so
# they live outside the offline `verify` chain. The slot is kept so CI
# can opt in once a kind-based fixture is added.
.PHONY: test-e2e
test-e2e: ## No-op e2e tests (would require a live OCM hub; see tests/README.md for the design)
	@echo "[flock-addon] no e2e tests yet; see tests/README.md"

.PHONY: test-chart
test-chart: lint test-unit ## Helm chart validation used by CI: lint + chart unit tests

.PHONY: image
image: ## No-op image target; images are built in source repositories
	@echo "[flock-addon] use source repositories to build images"

.PHONY: image-push
image-push: ## No-op image push target
	@echo "[flock-addon] use source repositories to push images"

.PHONY: image-manifest
image-manifest: ## No-op multi-arch manifest target
	@echo "[flock-addon] use source repositories to publish manifests"

##@ Development

.PHONY: lint
lint: ## Lint the Helm chart
	$(HELM) lint $(CHART_PATH)

# -----------------------------------------------------------------------------
# template
#
# Render twice: the default (Manual install strategy, no Placements) and
# the Placements path with both placements enabled. Together they exercise
# every conditional branch in the chart, so a template-only smoke test
# catches typos that helm lint alone would miss (e.g. a misnamed value
# referenced only inside `if .Values.placement.gpu.enabled`).
# -----------------------------------------------------------------------------

.PHONY: template
template: ## Render templates locally (useful for diffing across modes)
	@$(HELM) template $(RELEASE_NAME) $(CHART_PATH) >/dev/null
	@echo "[flock-addon] default render OK"
	@$(HELM) template $(RELEASE_NAME) $(CHART_PATH) \
		--set addon.installStrategy=Placements \
		--set placement.gpu.enabled=true \
		--set placement.all.enabled=true >/dev/null
	@echo "[flock-addon] Placements render OK"

##@ Help

.PHONY: help
help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-34s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)
