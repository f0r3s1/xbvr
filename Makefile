GORELEASER_CROSS_VERSION  ?= v1.24.5

SYSROOT_DIR     ?= sysroots
SYSROOT_ARCHIVE ?= sysroots.tar.bz2

# ── Dev targets (local development with hot reload) ────────────────

.PHONY: dev
dev: ## Build and run with hot reload (air + Vue HMR)
	DOCKER_BUILDKIT=1 docker compose -f docker-compose.dev.yml up --build

.PHONY: dev-build
dev-build: ## Build dev image only
	DOCKER_BUILDKIT=1 docker compose -f docker-compose.dev.yml build

.PHONY: dev-up
dev-up: ## Start dev without rebuilding
	docker compose -f docker-compose.dev.yml up

.PHONY: dev-down
dev-down: ## Stop dev containers
	docker compose -f docker-compose.dev.yml down

.PHONY: dev-logs
dev-logs: ## Tail dev container logs
	docker compose -f docker-compose.dev.yml logs -f

.PHONY: dev-shell
dev-shell: ## Shell into running dev container
	docker compose -f docker-compose.dev.yml exec xbvr-dev sh

.PHONY: dev-clean
dev-clean: ## Remove dev containers, volumes, and build cache
	docker compose -f docker-compose.dev.yml down -v
	docker builder prune -f --filter type=exec.cachemount

# ── Prod targets (CI / release builds) ──────────────────────────────

.PHONY: prod
prod: ## Build and run production image
	DOCKER_BUILDKIT=1 docker compose up --build

.PHONY: prod-build
prod-build: ## Production build - stateless, no cache mounts (CI-safe)
	DOCKER_BUILDKIT=1 docker build \
		--build-arg VERSION=$$(git describe --tags --always 2>/dev/null || echo "dev") \
		--build-arg COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown") \
		--build-arg DATE=$$(date -u +%Y-%m-%dT%H:%M:%SZ) \
		-t xbvr:latest .

# ── Sysroot targets ─────────────────────────────────────────────────

.PHONY: sysroot-pack
sysroot-pack:
	@tar cf - $(SYSROOT_DIR) -P | pv -s $[$(du -sk $(SYSROOT_DIR) | awk '{print $1}') * 1024] | pbzip2 > $(SYSROOT_ARCHIVE)

.PHONY: sysroot-unpack
sysroot-unpack:
	@pv $(SYSROOT_ARCHIVE) | pbzip2 -cd | tar -xf -

# ── GoReleaser targets ──────────────────────────────────────────────

.PHONY: release-dry-run-snapshot
release-dry-run-snapshot:
	@docker run \
		--rm \
		--privileged \
		-e CGO_ENABLED=1 \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src \
		-v `pwd`/sysroot:/sysroot \
		-w /go/src \
		ghcr.io/goreleaser/goreleaser-cross:${GORELEASER_CROSS_VERSION} \
		--clean --skip-validate --skip-publish --snapshot

.PHONY: release-dry-run
release-dry-run:
	@docker run \
		--rm \
		--privileged \
		-e CGO_ENABLED=1 \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src \
		-v `pwd`/sysroot:/sysroot \
		-w /go/src \
		ghcr.io/goreleaser/goreleaser-cross:${GORELEASER_CROSS_VERSION} \
		--clean --skip-validate --skip-publish

.PHONY: release-snapshot
release-snapshot:
	@if [ ! -f ".release-env" ]; then \
		echo "\033[91m.release-env is required for release\033[0m";\
		exit 1;\
	fi
	docker run \
		--rm \
		--privileged \
		-e CGO_ENABLED=1 \
		--env-file .release-env \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src \
		-v `pwd`/sysroot:/sysroot \
		-w /go/src \
		ghcr.io/goreleaser/goreleaser-cross:${GORELEASER_CROSS_VERSION} \
		release --clean --snapshot

.PHONY: release
release:
	@if [ ! -f ".release-env" ]; then \
		echo "\033[91m.release-env is required for release\033[0m";\
		exit 1;\
	fi
	docker run \
		--rm \
		--privileged \
		-e CGO_ENABLED=1 \
		--env-file .release-env \
		-v /var/run/docker.sock:/var/run/docker.sock \
		-v `pwd`:/go/src \
		-v `pwd`/sysroot:/sysroot \
		-w /go/src \
		ghcr.io/goreleaser/goreleaser-cross:${GORELEASER_CROSS_VERSION} \
		release --clean

# ── Help ─────────────────────────────────────────────────────────────

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
