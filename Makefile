# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

SHELL = /bin/bash
OS = $(shell uname | tr A-Z a-z)

# Project variables
PACKAGE = github.com/banzaicloud/pipeline
BINARY_NAME = pipeline
OPENAPI_DESCRIPTOR = docs/openapi/pipeline.yaml

# Build variables
BUILD_DIR ?= build
BUILD_PACKAGE = ${PACKAGE}/cmd/pipeline
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || git symbolic-ref -q --short HEAD)
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null)
BUILD_DATE ?= $(shell date +%FT%T%z)
LDFLAGS += -X main.version=${VERSION} -X main.commitHash=${COMMIT_HASH} -X main.buildDate=${BUILD_DATE}
export CGO_ENABLED ?= 0
ifeq (${VERBOSE}, 1)
	GOARGS += -v
endif

CLOUDINFO_VERSION = 0.7.0

GOLANGCI_VERSION = 1.17.1
MISSPELL_VERSION = 0.3.4
JQ_VERSION = 1.5
LICENSEI_VERSION = 0.1.0
OPENAPI_GENERATOR_VERSION = PR1869
MIGRATE_VERSION = 4.0.2
GOTESTSUM_VERSION = 0.3.2
GOBIN_VERSION = 0.0.10

GOLANG_VERSION = 1.13

GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./client/*")

.PHONY: up
up: config/dex.yml config/ui/feature-set.json start config/config.toml ## Set up the development environment

.PHONY: down
down: clean ## Destroy the development environment
	docker-compose down -v
	@ if [[ "$$OSTYPE" == "linux-gnu" ]]; then sudo rm -rf .docker/; else rm -rf .docker/; fi

.PHONY: reset
reset: down up ## Reset the development environment

.PHONY: clean
clean: ## Clean the working area and the project
	rm -rf bin/ ${BUILD_DIR}/
	rm -rf pipeline

docker-compose.override.yml: ## Create docker compose override file
	@ if [[ "$$OSTYPE" == "linux-gnu" ]]; then cat docker-compose.override.yml.dist | sed -e 's/# user: "$${uid}:$${gid}"/user: "$(shell id -u):$(shell id -g)"/' > docker-compose.override.yml; else cp docker-compose.override.yml.dist docker-compose.override.yml; fi

.PHONY: start
start: docker-compose.override.yml ## Start docker development environment
	@ if [ docker-compose.override.yml -ot docker-compose.override.yml.dist ]; then diff -u docker-compose.override.yml* || (echo "!!! The distributed docker-compose.override.yml example changed. Please update your file accordingly (or at least touch it). !!!" && false); fi
	mkdir -p .docker/volumes/{mysql,vault/file,vault/keys}
	docker-compose up -d

.PHONY: stop
stop: ## Stop docker development environment
	docker-compose stop

config/config.toml:
	cp config/config.toml.dist config/config.toml

config/ui/feature-set.json:
	mv config/ui/feature-set.json{,~} || true && cp config/ui/feature-set.json.dist config/ui/feature-set.json

config/dex.yml:
	cp config/dex.yml.dist config/dex.yml

.PHONY: run
run: GOTAGS += dev
run: build-pipeline ## Build and execute a binary
	PIPELINE_CONFIG_DIR=$${PWD}/config VAULT_ADDR="http://127.0.0.1:8200" ${BUILD_DIR}/${BINARY_NAME} ${ARGS}

.PHONY: run-worker
run-worker: GOTAGS += dev
run-worker: build-worker ## Build and execute a binary
	PIPELINE_CONFIG_DIR=$${PWD}/config VAULT_ADDR="http://127.0.0.1:8200" ${BUILD_DIR}/worker ${ARGS}

.PHONY: runall ## Run worker and pipeline in foreground. Use with make -j.
runall: run run-worker

.PHONY: goversion
goversion:
ifneq (${IGNORE_GOLANG_VERSION_REQ}, 1)
	@printf "${GOLANG_VERSION}\n$$(go version | awk '{sub(/^go/, "", $$3);print $$3}')" | sort -t '.' -k 1,1 -k 2,2 -k 3,3 -g | head -1 | grep -q -E "^${GOLANG_VERSION}$$" || (printf "Required Go version is ${GOLANG_VERSION}\nInstalled: `go version`" && exit 1)
endif

.PHONY: build-%
build-%: goversion ## Build a binary
	go build ${GOARGS} -tags "${GOTAGS}" -ldflags "${LDFLAGS}" -o ${BUILD_DIR}/$* ./cmd/$*

.PHONY: build
build: goversion ## Build all binaries
ifeq (${VERBOSE}, 1)
	go env
endif

	@mkdir -p ${BUILD_DIR}
	go build ${GOARGS} -tags "${GOTAGS}" -ldflags "${LDFLAGS}" -o ${BUILD_DIR}/ ./cmd/...

.PHONY: build-release
build-release: ## Build all binaries without debug information
	@${MAKE} LDFLAGS="-w ${LDFLAGS}" GOARGS="${GOARGS} -trimpath" BUILD_DIR="${BUILD_DIR}/release" build

.PHONY: build-debug
build-debug: ## Build all binaries with remote debugging capabilities
	@${MAKE} GOARGS="${GOARGS} -gcflags \"all=-N -l\"" BUILD_DIR="${BUILD_DIR}/debug" build

.PHONY: docker
docker: ## Build a Docker image
	@${MAKE} GOOS=linux GOARCH=amd64 build-release
	docker build -t banzaicloud/pipeline:local -f Dockerfile.local .

.PHONY: docker-debug
docker-debug: ## Build a Docker image with remote debugging capabilities
	@${MAKE} GOOS=linux GOARCH=amd64 build-debug
	docker build -t banzaicloud/pipeline:debug -f Dockerfile.debug .

.PHONY: check
check: test lint ## Run tests and linters

bin/golangci-lint: bin/golangci-lint-${GOLANGCI_VERSION}
	@ln -sf golangci-lint-${GOLANGCI_VERSION} bin/golangci-lint
bin/golangci-lint-${GOLANGCI_VERSION}:
	@mkdir -p bin
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | bash -s -- -b ./bin/ v${GOLANGCI_VERSION}
	@mv bin/golangci-lint $@

.PHONY: lint
lint: export CGO_ENABLED = 1
lint: bin/golangci-lint ## Run linter
	bin/golangci-lint run

.PHONY: fmt
fmt:
	@gofmt -s -w ${GOFILES_NOVENDOR}

bin/misspell: bin/misspell-${MISSPELL_VERSION}
	@ln -sf misspell-${MISSPELL_VERSION} bin/misspell
bin/misspell-${MISSPELL_VERSION}:
	@mkdir -p bin
	curl -sfL https://git.io/misspell | bash -s -- -b ./bin/ v${MISSPELL_VERSION}
	@mv bin/misspell $@

.PHONY: misspell
misspell: bin/misspell ## Fix spelling mistakes
	misspell -w ${GOFILES_NOVENDOR}

bin/licensei: bin/licensei-${LICENSEI_VERSION}
	@ln -sf licensei-${LICENSEI_VERSION} bin/licensei
bin/licensei-${LICENSEI_VERSION}:
	@mkdir -p bin
	curl -sfL https://raw.githubusercontent.com/goph/licensei/master/install.sh | bash -s v${LICENSEI_VERSION}
	@mv bin/licensei $@

.PHONY: license-check
license-check: bin/licensei ## Run license check
	bin/licensei check
	./scripts/check-header.sh

.PHONY: license-cache
license-cache: bin/licensei ## Generate license cache
	bin/licensei cache

bin/gotestsum: bin/gotestsum-${GOTESTSUM_VERSION}
	@ln -sf gotestsum-${GOTESTSUM_VERSION} bin/gotestsum
bin/gotestsum-${GOTESTSUM_VERSION}:
	@mkdir -p bin
	curl -L https://github.com/gotestyourself/gotestsum/releases/download/v${GOTESTSUM_VERSION}/gotestsum_${GOTESTSUM_VERSION}_${OS}_amd64.tar.gz | tar -zOxf - gotestsum > ./bin/gotestsum-${GOTESTSUM_VERSION} && chmod +x ./bin/gotestsum-${GOTESTSUM_VERSION}

.PHONY: test
TEST_PKGS ?= ./...
TEST_REPORT_NAME ?= results.xml
.PHONY: test
test: TEST_REPORT ?= main
test: SHELL = /bin/bash
test: export CGO_ENABLED = 1
test: bin/gotestsum ## Run tests
	@mkdir -p ${BUILD_DIR}/test_results/${TEST_REPORT}
	bin/gotestsum --no-summary=skipped --junitfile ${BUILD_DIR}/test_results/${TEST_REPORT}/${TEST_REPORT_NAME} -- $(filter-out -v,${GOARGS})  $(if ${TEST_PKGS},${TEST_PKGS},./...)

.PHONY: test-all
test-all: ## Run all tests
	@${MAKE} GOARGS="${GOARGS} -run .\*" TEST_REPORT=all test

.PHONY: test-integration
test-integration: ## Run integration tests
	@${MAKE} GOARGS="${GOARGS} -run ^TestIntegration\$$\$$" TEST_REPORT=integration test

bin/gobin: bin/gobin-${GOBIN_VERSION}
	@ln -sf gobin-${GOBIN_VERSION} bin/gobin
bin/gobin-${GOBIN_VERSION}:
	@mkdir -p bin
	curl -L https://github.com/myitcv/gobin/releases/download/v${GOBIN_VERSION}/${OS}-amd64 > ./bin/gobin-${GOBIN_VERSION} && chmod +x ./bin/gobin-${GOBIN_VERSION}

bin/mockery: bin/gobin
	@mkdir -p bin
	GOBIN=bin/ bin/gobin github.com/vektra/mockery/cmd/mockery

.PHONY: generate-mocks
generate-mocks: bin/mockery ## Generate mocks
	MOCKERY=$(abspath bin/mockery) go generate ./...

.PHONY: validate-openapi
validate-openapi: ## Validate the openapi description
	docker run --rm -v $${PWD}:/local banzaicloud/openapi-generator-cli:${OPENAPI_GENERATOR_VERSION} validate --recommend -i /local/${OPENAPI_DESCRIPTOR}

.PHONY: generate-client
generate-client: validate-openapi ## Generate go client based on openapi description
	@ if [[ "$$OSTYPE" == "linux-gnu" ]]; then sudo rm -rf ./client; else rm -rf ./client/; fi
	docker run --rm -v $${PWD}:/local banzaicloud/openapi-generator-cli:${OPENAPI_GENERATOR_VERSION} generate \
	--additional-properties packageName=client \
	--additional-properties withGoCodegenComment=true \
	-i /local/${OPENAPI_DESCRIPTOR} \
	-g go \
	-o /local/client
	@ if [[ "$$OSTYPE" == "linux-gnu" ]]; then sudo chown -R $(shell id -u):$(shell id -g) client/; fi
	gofmt -s -w client/

ifeq (${OS}, darwin)
	shasum -a 256 ${OPENAPI_DESCRIPTOR} > client/SHA256SUMS
endif
ifeq (${OS}, linux)
	sha256sum ${OPENAPI_DESCRIPTOR} > client/SHA256SUMS
endif

bin/migrate: bin/migrate-${MIGRATE_VERSION}
	@ln -sf migrate-${MIGRATE_VERSION} bin/migrate
bin/migrate-${MIGRATE_VERSION}:
	@mkdir -p bin
	curl -L https://github.com/golang-migrate/migrate/releases/download/v${MIGRATE_VERSION}/migrate.${OS}-amd64.tar.gz | tar xvz -C bin
	@mv bin/migrate.${OS}-amd64 $@

.PHONY: generate-cloudinfo-client
generate-cloudinfo-client: ## Generate client from Cloudinfo OpenAPI spec
	curl https://raw.githubusercontent.com/banzaicloud/cloudinfo/${CLOUDINFO_VERSION}/api/openapi-spec/cloudinfo.yaml | sed "s/version: .*/version: ${CLOUDINFO_VERSION}/" > cloudinfo-openapi.yaml
	rm -rf .gen/cloudinfo
	docker run --rm -v ${PWD}:/local banzaicloud/openapi-generator-cli:${OPENAPI_GENERATOR_VERSION} generate \
	--additional-properties packageName=cloudinfo \
	--additional-properties withGoCodegenComment=true \
	-i /local/cloudinfo-openapi.yaml \
	-g go \
	-o /local/.gen/cloudinfo
	rm cloudinfo-openapi.yaml .gen/cloudinfo/.travis.yml .gen/cloudinfo/git_push.sh

.PHONY: list
list: ## List all make targets
	@$(MAKE) -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort

.PHONY: help
.DEFAULT_GOAL := help
help:
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# Variable outputting/exporting rules
var-%: ; @echo $($*)
varexport-%: ; @echo $*=$($*)
