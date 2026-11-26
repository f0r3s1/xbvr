GORELEASER_CROSS_VERSION  ?= v1.27.3

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
BUILD_VERSION ?= $(shell git describe --tags --always 2>/dev/null || echo dev)

prod-build: ## Production build - stateless, no cache mounts (CI-safe)
	DOCKER_BUILDKIT=1 docker build \
		--build-arg VERSION=$(BUILD_VERSION) \
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

# ── Lint / Format / Test ────────────────────────────────────────────

.PHONY: lint
lint: lint-go lint-js lint-actions ## Run all linters (Go + JS + GitHub workflows)

.PHONY: lint-actions
lint-actions: ## Lint GitHub workflow files (actionlint + shellcheck) in a self-deleting docker container
	@docker run --rm -v "$$PWD":/repo -w /repo rhysd/actionlint:1.7.12

.PHONY: lint-go
lint-go: ## Run Go linters in a self-deleting docker container (no local tools needed)
	@mkdir -p ui/dist && touch ui/dist/embed_placeholder
	@docker run --rm \
		-v "$$PWD":/app -w /app \
		-v xbvr-go-mod:/go/pkg/mod \
		-v xbvr-go-build:/root/.cache/go-build \
		golang:1.27-bookworm sh -c '\
			set -e; \
			apt-get update -qq >/dev/null; \
			apt-get install -y -qq pkg-config libayatana-appindicator3-dev libgtk-3-dev >/dev/null; \
			echo "→ gofmt"; \
			out=$$(gofmt -l . | grep -v "^vendor/" || true); if [ -n "$$out" ]; then echo "$$out"; exit 1; fi; \
			echo "→ go vet"; go vet ./...; \
			echo "→ golangci-lint"; \
			go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest; \
			$$(go env GOPATH)/bin/golangci-lint run'

.PHONY: lint-js
lint-js: ## Run JS/Vue linter in a self-deleting docker container
	@docker run --rm \
		-v "$$PWD":/app -w /app/ui \
		-v xbvr-bun-cache:/root/.bun/install/cache \
		oven/bun:latest sh -c 'bun install --frozen-lockfile --ignore-scripts && bunx eslint src/'

.PHONY: ui-build
ui-build: ## Build the Vue/Vite UI bundle in a self-deleting docker container (no host tools needed)
	@docker run --rm \
		-v "$$PWD":/app -w /app \
		-v xbvr-bun-cache:/root/.bun/install/cache \
		oven/bun:latest sh -c 'bun install --frozen-lockfile --ignore-scripts && bun run build'

.PHONY: lint-fix
lint-fix: ## Auto-fix lint issues where possible (docker, self-deleting)
	@docker run --rm -v "$$PWD":/app -w /app golang:1.27-bookworm gofmt -s -w .
	@docker run --rm -v "$$PWD":/app -w /app/ui oven/bun:latest sh -c 'bun install --frozen-lockfile --ignore-scripts && bunx eslint src/ --fix'

.PHONY: fmt
fmt: lint-fix ## Alias of lint-fix

GO_IMAGE ?= golang:1.27-bookworm
GO_DOCKER = docker run --rm \
		-v "$$PWD":/app -w /app \
		-v xbvr-go-mod:/go/pkg/mod \
		-v xbvr-go-build:/root/.cache/go-build \
		$(GO_IMAGE) sh -c
GO_CDEPS = apt-get update -qq >/dev/null; apt-get install -y -qq pkg-config libayatana-appindicator3-dev libgtk-3-dev >/dev/null 2>&1

.PHONY: test
test: ## Build, vet and run Go tests in a self-deleting docker container
	@mkdir -p ui/dist && touch ui/dist/embed_placeholder
	@$(GO_DOCKER) 'set -e; $(GO_CDEPS); \
		go build -tags=json1 ./... && go test -tags=json1 ./pkg/...'

.PHONY: test-live
test-live: ## Live scraper tests against real sites; needs XBVR_SCRAPE_TARGETS=/path/to/targets.json
	@test -n "$(XBVR_SCRAPE_TARGETS)" || { echo "set XBVR_SCRAPE_TARGETS=/path/to/targets.json"; exit 1; }
	@mkdir -p ui/dist && touch ui/dist/embed_placeholder
	@docker run --rm -v "$$PWD":/app -w /app -v xbvr-go-mod:/go/pkg/mod -v xbvr-go-build:/root/.cache/go-build \
		-v "$(XBVR_SCRAPE_TARGETS)":/targets.json:ro -e XBVR_SCRAPE_TARGETS=/targets.json \
		$(GO_IMAGE) go test -tags='json1 scrapelive' ./pkg/scrape/ -run TestLiveScrape -v

.PHONY: deps-update
deps-update: ## Upgrade all Go modules and JS packages to latest (docker), then tidy
	@$(GO_DOCKER) 'set -e; $(GO_CDEPS); go get -u ./... ; go mod tidy'
	@docker run --rm -v "$$PWD":/app -w /app -v xbvr-bun-cache:/root/.bun/install/cache \
		oven/bun:latest sh -c 'bun update --latest && bun outdated'

# ── Local seeded instance (SFW dummy data for UI work / Playwright) ──
# `make local` = build + run + seed. Data lives in two named volumes, so the
# instance survives restarts; `make local-reset` wipes it for a clean start.

LOCAL_NAME   ?= xbvr-local
LOCAL_CONFIG ?= xbvr-local-config
LOCAL_MEDIA  ?= xbvr-local-media
LOCAL_ENV    ?=

.PHONY: local
local: local-up local-seed ## Build, run and seed a local instance on http://localhost:9999

.PHONY: local-up
local-up: BUILD_VERSION = latest
local-up: prod-build ## Build the prod image and (re)start the local instance
	@docker rm -f $(LOCAL_NAME) >/dev/null 2>&1 || true
	@docker run -d --name $(LOCAL_NAME) -p 9999:9999 -p 9998:9998 $(LOCAL_ENV) \
		-v $(LOCAL_CONFIG):/root/.config -v $(LOCAL_MEDIA):/media xbvr:latest >/dev/null
	@for i in $$(seq 1 60); do curl -sf -o /dev/null http://localhost:9999/ && break; sleep 1; done
	@echo "XBVR running at http://localhost:9999"

.PHONY: local-seed
local-seed: ## Load SFW dummy scenes, sample videos and funscripts into the local instance
	@docker run --rm -v "$$PWD/scripts":/s:ro -v $(LOCAL_MEDIA):/media \
		-e SCENES=$${SCENES:-240} -e PEOPLE=$${PEOPLE:-48} oven/bun:latest bun /s/seed-dev-data.mjs

.PHONY: local-logs
local-logs: ## Tail the local instance logs
	@docker logs -f $(LOCAL_NAME)

.PHONY: local-reset
local-reset: ## Remove the local instance and wipe its data and media volumes
	@docker rm -f $(LOCAL_NAME) >/dev/null 2>&1 || true
	@docker volume rm $(LOCAL_CONFIG) $(LOCAL_MEDIA) >/dev/null 2>&1 || true

.PHONY: vuln
vuln: ## Run security scanners (govulncheck + bun audit) in self-deleting docker containers
	@mkdir -p ui/dist && touch ui/dist/embed_placeholder
	@$(GO_DOCKER) 'go run golang.org/x/vuln/cmd/govulncheck@latest $$(go list -e ./... | grep -v "/pkg/tray$$")'
	@docker run --rm -v "$$PWD":/app -w /app -v xbvr-bun-cache:/root/.bun/install/cache \
		oven/bun:latest sh -c 'bun install --frozen-lockfile --ignore-scripts >/dev/null && bun audit'

# ── Help ─────────────────────────────────────────────────────────────

.PHONY: help
help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
