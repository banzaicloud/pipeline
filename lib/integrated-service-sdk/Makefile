
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

OS = $(shell uname | tr A-Z a-z)

CONTROLLER_GEN_VERSION = v0.2.5
CONTROLLER_GEN = $(PWD)/bin/controller-gen

GOLANGCI_VERSION = 1.30.0
LICENSEI_VERSION = 0.3.1

E2E_TESTS = $(shell ls -d e2e/*/)

export PATH := $(PWD)/bin:$(PATH)

# Run tests
test: generate manifests ## Run unit tests
	go test ./... -coverprofile cover.out

# Install CRDs into a cluster
install: manifests ## Install CRDs
	kubectl apply -f config/crd/bases

# Uninstall CRDs from a cluster
uninstall: manifests ## Uninstall CRDs
	kubectl delete -f config/crd/bases

# Generate manifests e.g. CRD, RBAC etc.
manifests: bin/controller-gen ## Generate manifests
	$(CONTROLLER_GEN) $(CRD_OPTIONS) webhook paths="./..." output:crd:artifacts:config=config/crd/bases

bin/golangci-lint: bin/golangci-lint-${GOLANGCI_VERSION}
	@ln -sf golangci-lint-${GOLANGCI_VERSION} bin/golangci-lint
bin/golangci-lint-${GOLANGCI_VERSION}:
	@mkdir -p bin
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | bash -s -- -b ./bin/ v${GOLANGCI_VERSION}
	@mv bin/golangci-lint $@

lint-fix: export CGO_ENABLED = 1
lint-fix: bin/golangci-lint ## Fix lint violations
	gofmt -w -s .
	bin/golangci-lint run --fix

lint: export CGO_ENABLED = 1
lint: bin/golangci-lint ## Run go fmt, go vet, golangci lint
	go fmt ./...
	go vet ./...

helm-lint:
	helm lint charts/integrated-service-operator

bin/licensei: bin/licensei-${LICENSEI_VERSION}
	@ln -sf licensei-${LICENSEI_VERSION} bin/licensei
bin/licensei-${LICENSEI_VERSION}:
	@mkdir -p bin
	curl -sfL https://git.io/licensei | bash -s v${LICENSEI_VERSION}
	@mv bin/licensei $@

.PHONY: license-check
license-check: bin/licensei ## Run license check
	bin/licensei check
	bin/licensei header

.PHONY: license-cache
license-cache: bin/licensei ## Generate license cache
	bin/licensei cache

# Generate code
generate: bin/controller-gen ## Generate code
	$(CONTROLLER_GEN) object paths="./..."

.PHONY: bin/controller-gen
bin/controller-gen:
	@ if ! test -x bin/controller-gen; then \
		set -ex ;\
		CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
		cd $$CONTROLLER_GEN_TMP_DIR ;\
		go mod init tmp ;\
		GOBIN=$(PWD)/bin go get sigs.k8s.io/controller-tools/cmd/controller-gen@${CONTROLLER_GEN_VERSION} ;\
		rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	fi

tidy: ## Run go mod tidy on the root and nested projects
	@find . -iname "go.mod" | xargs -L1 sh -c 'cd $$(dirname $$0); go mod tidy'

check: tidy license-cache license-check test ## Run generators, linters, license checks and tests and check for uncommitted changes
	git diff --exit-code

.PHONY: list
list: ## List all make targets
	@$(MAKE) -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort

.PHONY: help
.DEFAULT_GOAL := help
help:
	@grep -h -E '^[/a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

