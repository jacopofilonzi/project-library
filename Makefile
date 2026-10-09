# Shortcuts for the commands in AGENTS.md. `make help` lists them.
# The Wails tasks themselves live in Taskfile.yml: this file only wraps them.

# On Windows the recipes run in Git for Windows' sh (with grep, awk…), also
# when make is started from PowerShell or cmd. Short path: make does not like spaces.
ifeq ($(OS),Windows_NT)
  SHELL := $(or $(GIT_SH),C:/PROGRA~1/Git/bin/sh.exe)
else
  SHELL := sh
endif

GOPATH := $(shell go env GOPATH)
HOSTARCH := $(shell go env GOARCH)

# A 32-bit Go cannot run Wails: build and test as amd64.
ifeq ($(HOSTARCH),386)
  ARCH ?= amd64
else
  ARCH ?= $(HOSTARCH)
endif
export GOARCH := $(ARCH)

# The Wails tasks call wails3 by name: put GOPATH/bin in the PATH (and
# bin/windows_amd64, where it ends up when Go is 32-bit), plus makensis on Windows.
ifeq ($(OS),Windows_NT)
  GOBIN_SH := $$(cygpath -u '$(GOPATH)')/bin
  WAILS := PATH="$(GOBIN_SH):$(GOBIN_SH)/windows_amd64:/c/Program Files (x86)/NSIS:$$PATH" wails3
else
  WAILS := PATH="$(GOPATH)/bin:$$PATH" wails3
endif

PLATFORM_PKGS = $$(go list ./internal/... | grep -v /internal/core)

.DEFAULT_GOAL := help
.PHONY: help deps dev build package run bindings test vet vet-cross check e2e verify version forge-live clean

help: ## Show this list
	@grep -E '^[a-z0-9-]+:.*## ' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-11s\033[0m %s\n", $$1, $$2}'

deps: ## Install the frontend dependencies
	cd frontend && npm install

dev: ## Run the app with hot reload
	$(WAILS) dev

build: ## Build the app in bin/
	$(WAILS) build ARCH=$(ARCH)

package: ## Build the installer in bin/
	$(WAILS) package ARCH=$(ARCH)

run: build ## Build and start the app
	$(WAILS) task run ARCH=$(ARCH)

bindings: ## Regenerate frontend/bindings after changing the methods of Library
	$(WAILS) generate bindings -clean=true -ts -i

test: ## Go tests
	go test ./internal/...

vet: ## go vet, for this system and for internal/platform on the others
	go vet ./...
	$(MAKE) --no-print-directory vet-cross

vet-cross: ## go vet of the packages without Wails for macOS and Linux
	GOOS=darwin go vet $(PLATFORM_PKGS)
	GOOS=linux go vet $(PLATFORM_PKGS)

check: ## svelte-check (must give 0 errors and 0 warnings)
	cd frontend && npx svelte-check --tsconfig ./tsconfig.json --fail-on-warnings

e2e: ## Playwright tests with the fake backend
	cd frontend && npm run test:e2e

verify: test vet check e2e build ## Everything that must pass after a change

version: ## Set the version everywhere: make version V=x.y.z
	@test -n "$(V)" || { echo "usage: make version V=x.y.z"; exit 1; }
	sh scripts/set-version.sh $(V)

forge-live: ## Live read-only forge test: make forge-live CLI=gh REPO=owner/name
	@test -n "$(REPO)" || { echo "usage: make forge-live CLI=gh|glab REPO=owner/name"; exit 1; }
	PL_FORGE_LIVE=$(or $(CLI),gh) PL_FORGE_REPO=$(REPO) go test ./internal/forge -run TestLive -v

clean: ## Remove the build output
	rm -rf bin frontend/dist
