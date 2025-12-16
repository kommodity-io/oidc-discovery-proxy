VERSION			?= $(shell git describe --tags --always)
TREE_STATE      ?= $(shell git describe --always --dirty --exclude='*' | grep -q dirty && echo dirty || echo clean)
COMMIT			?= $(shell git rev-parse HEAD)
BUILD_DATE		?= $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
GO_FLAGS		:= -ldflags "-X 'k8s.io/component-base/version.gitVersion=$(VERSION)' -X 'k8s.io/component-base/version.gitTreeState=$(TREE_STATE)' -X 'k8s.io/component-base/version.buildDate=$(BUILD_DATE)' -X 'k8s.io/component-base/version.gitCommit=$(COMMIT)'"
SOURCES			:= $(shell find . -name '*.go')
UPX_FLAGS		?= -qq

##@ General

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Dependencies

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Set up the linter.
LINTER := bin/golangci-lint

.PHONY: golangci-lint
golangci-lint: $(LINTER) ## Download golangci-lint locally if necessary.
$(LINTER):
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b bin/ v2.4.0

lint: $(LINTER) ## Run the linter.
	$(LINTER) run

lint-fix: $(LINTER) ## Run the linter and fix issues.
	$(LINTER) run --fix

build: $(SOURCES) ## Build the application.
	go build $(GO_FLAGS) -o bin/oidc-discovery-proxy cmd/oidc-discovery-proxy/main.go
ifneq ($(UPX_FLAGS),)
	upx $(UPX_FLAGS) bin/oidc-discovery-proxy
endif

build-image: ## Build the Container image.
	docker buildx build \
	-f Containerfile \
	-t oidc-discovery-proxy:latest \
	. \
	--build-arg VERSION=$(VERSION) \
	--load

run: ## Run the application locally.
	LOG_FORMAT=console \
	LOG_LEVEL=info \
	go run $(GO_FLAGS) cmd/oidc-discovery-proxy/main.go
