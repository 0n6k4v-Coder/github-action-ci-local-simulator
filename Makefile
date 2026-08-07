.PHONY: all help test test-unit test-integration build vet fmt lint clean audit clean-docker clean-all check-go check-docker

# ============================================================================
# Universal Go Makefile
# Works on Linux, macOS, Windows (WSL2/Git Bash)
# Auto-detects Go binary, falls back to Docker if not found
# ============================================================================

# --- Binary detection --------------------------------------------------------
GO_PATH_BIN := $(shell command -v go 2>/dev/null)

ifeq ($(GO_PATH_BIN),)
GO_PATH_BIN := $(shell [ -x /usr/local/go/bin/go ] && echo /usr/local/go/bin/go)
endif
ifeq ($(GO_PATH_BIN),)
GO_PATH_BIN := $(shell [ -x $(HOME)/go/bin/go ] && echo $(HOME)/go/bin/go)
endif
ifeq ($(GO_PATH_BIN),)
GO_PATH_BIN := $(shell [ -x /usr/bin/go ] && echo /usr/bin/go)
endif
ifeq ($(GO_PATH_BIN),)
GO_PATH_BIN := $(shell [ -x /opt/go/bin/go ] && echo /opt/go/bin/go)
endif
ifeq ($(GO_PATH_BIN),)
GO_PATH_BIN := $(shell [ -x /c/Program\ Files/Go/bin/go.exe ] && echo /c/Program\ Files/Go/bin/go.exe)
endif

DOCKER_BIN := $(shell command -v docker 2>/dev/null)
DOCKER_IMAGE ?= golang:1.26
DOCKER_WORKDIR ?= /workspace
DOCKER_SOCKET ?= /var/run/docker.sock
ifeq ($(OS),Windows_NT)
DOCKER_SOCKET := //var/run/docker.sock
endif

DOCKER_RUN := $(DOCKER_BIN) run --rm \
	-v "$(CURDIR):$(DOCKER_WORKDIR)" \
	-v "$(DOCKER_SOCKET):/var/run/docker.sock" \
	-w "$(DOCKER_WORKDIR)" \
	$(DOCKER_IMAGE)

# --- Runtime selection -------------------------------------------------------
ifeq ($(GO_PATH_BIN),)
ifeq ($(DOCKER_BIN),)
GO := false
GO_SOURCE := unavailable
else
GO := $(DOCKER_RUN) go
GO_SOURCE := docker
endif
else
GO := $(GO_PATH_BIN)
GO_SOURCE := local
endif

# --- Configurable flags ------------------------------------------------------
GO_TEST_TIMEOUT ?= 10m
GO_TEST_INTEGRATION_TIMEOUT ?= 10m
GO_TEST_PKGS ?= ./...
GO_BUILD_PKGS ?= ./...
GO_VET_PKGS ?= ./...
GO_INTEGRATION_TAGS ?= integration
GOFLAGS ?=
STRICT_MISSING_RUNTIME ?= 0

define SKIP_OR_FAIL_IF_NO_RUNTIME
	@if [ "$(GO_SOURCE)" = "unavailable" ]; then \
		if [ "$(STRICT_MISSING_RUNTIME)" = "1" ]; then \
			echo "Error: Neither Go nor Docker found. Install Go: https://go.dev/doc/install"; \
			exit 1; \
		else \
			echo "Skipping '$@': neither Go nor Docker found. Set STRICT_MISSING_RUNTIME=1 to fail."; \
		fi; \
	fi
endef

define RUN_GO_OR_SKIP
	@if [ "$(GO_SOURCE)" = "unavailable" ]; then \
		if [ "$(STRICT_MISSING_RUNTIME)" = "1" ]; then \
			echo "Error: Neither Go nor Docker found. Install Go: https://go.dev/doc/install"; \
			exit 1; \
		else \
			echo "Skipping '$@': neither Go nor Docker found. Set STRICT_MISSING_RUNTIME=1 to fail."; \
		fi; \
	else \
		$(1); \
	fi
endef

help:
	@echo "Universal Go Makefile"
	@echo ""
	@echo "Go source: $(GO_SOURCE)"
	@echo "Go path: $(if $(GO_PATH_BIN),$(GO_PATH_BIN),not found)"
	@echo "Docker path: $(if $(DOCKER_BIN),$(DOCKER_BIN),not found)"
	@echo ""
	@echo "Targets:"
	@echo "  make test             Run all tests"
	@echo "  make test-unit        Run unit tests (short mode)"
	@echo "  make test-integration Run integration tests (-tags $(GO_INTEGRATION_TAGS))"
	@echo "  make build            Build packages"
	@echo "  make vet              Run go vet"
	@echo "  make fmt              Format code with gofmt"
	@echo "  make lint             Run golangci-lint when available"
	@echo "  make clean            Remove common build artifacts"
	@echo "  make check-go         Show selected Go runtime"
	@echo "  make check-docker     Show Docker status"
	@echo "  make audit            Show current Docker waste"
	@echo "  make clean-docker     Light cleanup (removes stopped containers, dangling images, test artifacts)"
	@echo "  make clean-all        Full cleanup (removes all unused Docker resources between phases)"
	@echo "  make clean-all PRUNE_IMAGES=1   .. also removes ALL unused images"
	@echo ""
	@echo "Options:"
	@echo "  STRICT_MISSING_RUNTIME=1  Fail when Go and Docker are both unavailable"

check-go:
	@echo "=== Go Environment ==="
	@echo "Go source: $(GO_SOURCE)"
	@echo "Go command: $(GO)"
	@if [ "$(GO_SOURCE)" = "unavailable" ]; then \
		if [ "$(STRICT_MISSING_RUNTIME)" = "1" ]; then \
			echo "Error: Neither Go nor Docker found. Install Go: https://go.dev/doc/install"; \
			exit 1; \
		else \
			echo "Skipping version check: neither Go nor Docker found."; \
		fi; \
	else \
		$(GO) version; \
	fi

check-docker:
	@echo "=== Docker Environment ==="
	@if [ -n "$(DOCKER_BIN)" ]; then \
		echo "Docker: available"; \
		"$(DOCKER_BIN)" --version; \
		echo "Docker socket: $(DOCKER_SOCKET)"; \
		ls -la "$(DOCKER_SOCKET)" 2>/dev/null || echo "Warning: Docker socket not found"; \
	else \
		echo "Docker: NOT available"; \
	fi

test: check-go
	$(call RUN_GO_OR_SKIP,$(GO) test $(GOFLAGS) $(GO_TEST_PKGS) -timeout $(GO_TEST_TIMEOUT))

test-unit: check-go
	$(call RUN_GO_OR_SKIP,$(GO) test $(GOFLAGS) -short $(GO_TEST_PKGS) -timeout $(GO_TEST_TIMEOUT))

test-integration: check-go check-docker
	@$(call SKIP_OR_FAIL_IF_NO_RUNTIME)
	@if [ "$(GO_SOURCE)" = "unavailable" ]; then :; \
	elif [ -n "$(DOCKER_BIN)" ]; then \
		$(GO) test $(GOFLAGS) -tags "$(GO_INTEGRATION_TAGS)" $(GO_TEST_PKGS) -timeout $(GO_TEST_INTEGRATION_TIMEOUT); \
	else \
		echo "Skipping integration tests (Docker not available)"; \
	fi

build: check-go
	$(call RUN_GO_OR_SKIP,$(GO) build $(GOFLAGS) $(GO_BUILD_PKGS))

vet: check-go
	$(call RUN_GO_OR_SKIP,$(GO) vet $(GOFLAGS) $(GO_VET_PKGS))

fmt: check-go
	$(call RUN_GO_OR_SKIP,$(GO) fmt $(GO_TEST_PKGS))

lint: check-go
	@if [ "$(GO_SOURCE)" = "unavailable" ]; then \
		if [ "$(STRICT_MISSING_RUNTIME)" = "1" ]; then \
			echo "Error: Neither Go nor Docker found. Install Go: https://go.dev/doc/install"; \
			exit 1; \
		else \
			echo "Skipping '$@': neither Go nor Docker found. Set STRICT_MISSING_RUNTIME=1 to fail."; \
		fi; \
	elif command -v golangci-lint >/dev/null 2>&1; then \
		golangci-lint run; \
	else \
		echo "golangci-lint not found, falling back to go vet"; \
		$(GO) vet $(GOFLAGS) $(GO_VET_PKGS); \
	fi

clean:
	@rm -rf ./bin ./dist

## audit: Show current Docker waste
audit:
	@bash scripts/audit-waste.sh

## clean-docker: Light cleanup (safe for frequent use)
clean-docker:
	@bash scripts/cleanup-light.sh

## clean-all: Full cleanup (between development phases)
##   Usage: make clean-all                # Safe mode (preserves recent images)
##   Usage: make clean-all PRUNE_IMAGES=1 # Prune images mode (removes ALL images)
clean-all:
	@bash scripts/cleanup-full.sh $(if $(PRUNE_IMAGES),--prune-images)

all: fmt vet test
