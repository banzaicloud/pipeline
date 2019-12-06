# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

SHELL = /bin/bash
OS = $(shell uname | tr A-Z a-z)
export PATH := $(abspath bin/):${PATH}

# Project variables
PACKAGE = github.com/banzaicloud/pipeline
BINARY_NAME = pipeline
OPENAPI_DESCRIPTOR = apis/pipeline/pipeline.yaml

# Build variables
BUILD_DIR ?= build
BUILD_PACKAGE = ${PACKAGE}/cmd/pipeline
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || git symbolic-ref -q --short HEAD)
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null)
BUILD_DATE ?= $(shell date +%FT%T%z)
LDFLAGS += -X main.version=${VERSION} -X main.commitHash=${COMMIT_HASH} -X main.buildDate=${BUILD_DATE}
export CGO_ENABLED ?= 0
ifeq (${VERBOSE}, 1)
ifeq ($(filter -v,${GOARGS}),)
	GOARGS += -v
endif
TEST_FORMAT = short-verbose
endif

CLOUDINFO_VERSION = 0.7.0
DEX_VERSION = 2.19.0
# TODO: use an exact version
ANCHORE_VERSION = 156836d

GOLANGCI_VERSION = 1.21.0
JQ_VERSION = 1.5
LICENSEI_VERSION = 0.2.0
OPENAPI_GENERATOR_VERSION = v4.1.3
MIGRATE_VERSION = 4.0.2
GOTESTSUM_VERSION = 0.3.5
GOBIN_VERSION = 0.0.13
PROTOTOOL_VERSION = 1.8.0
PROTOC_GEN_GO_VERSION = 1.3.2
MGA_VERSION = 0.0.11

GOLANG_VERSION = 1.13

.PHONY: up
up: etc/config/dex.yml config/ui/feature-set.json start config/config.yaml ## Set up the development environment

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

config/config.yaml:
	cp config/config.dev.yaml config/config.yaml

config/ui/feature-set.json:
	mv config/ui/feature-set.json{,~} || true && cp config/ui/feature-set.json.dist config/ui/feature-set.json

etc/config/dex.yml:
	cp etc/config/dex.yml.dist etc/config/dex.yml

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

.PHONY: fix
fix: export CGO_ENABLED = 1
fix: bin/golangci-lint ## Fix lint violations
	bin/golangci-lint run --fix

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
test: TEST_FORMAT ?= short
test: SHELL = /bin/bash
test: export CGO_ENABLED = 1
test: bin/gotestsum ## Run tests
	@mkdir -p ${BUILD_DIR}/test_results/${TEST_REPORT}
	bin/gotestsum --no-summary=skipped --junitfile ${BUILD_DIR}/test_results/${TEST_REPORT}/${TEST_REPORT_NAME} --format ${TEST_FORMAT} -- $(filter-out -v,${GOARGS})  $(if ${TEST_PKGS},${TEST_PKGS},./...)

.PHONY: test-all
test-all: ## Run all tests
	@${MAKE} GOARGS="${GOARGS} -run .\*" TEST_REPORT=all test

.PHONY: test-integration
test-integration: bin/test/kube-apiserver bin/test/etcd ## Run integration tests
	@${MAKE} TEST_ASSET_KUBE_APISERVER=$(abspath bin/test/kube-apiserver) TEST_ASSET_ETCD=$(abspath bin/test/etcd) GOARGS="${GOARGS} -run ^TestIntegration\$$\$$" TEST_REPORT=integration test

bin/test/kube-apiserver:
	@mkdir -p bin/test
	curl -L https://storage.googleapis.com/k8s-c10s-test-binaries/kube-apiserver-$(shell uname)-x86_64 > bin/test/kube-apiserver
	chmod +x bin/test/kube-apiserver

bin/test/etcd:
	@mkdir -p bin/test
	curl -L https://storage.googleapis.com/k8s-c10s-test-binaries/etcd-$(shell uname)-x86_64 > bin/test/etcd
	chmod +x bin/test/etcd

bin/migrate: bin/migrate-${MIGRATE_VERSION}
	@ln -sf migrate-${MIGRATE_VERSION} bin/migrate
bin/migrate-${MIGRATE_VERSION}:
	@mkdir -p bin
	curl -L https://github.com/golang-migrate/migrate/releases/download/v${MIGRATE_VERSION}/migrate.${OS}-amd64.tar.gz | tar xvz -C bin
	@mv bin/migrate.${OS}-amd64 $@

bin/gobin: bin/gobin-${GOBIN_VERSION}
	@ln -sf gobin-${GOBIN_VERSION} bin/gobin
bin/gobin-${GOBIN_VERSION}:
	@mkdir -p bin
	curl -L https://github.com/myitcv/gobin/releases/download/v${GOBIN_VERSION}/${OS}-amd64 > ./bin/gobin-${GOBIN_VERSION} && chmod +x ./bin/gobin-${GOBIN_VERSION}

bin/mga: bin/mga-${MGA_VERSION}
	@ln -sf mga-${MGA_VERSION} bin/mga
bin/mga-${MGA_VERSION}:
	@mkdir -p bin
	curl -sfL https://git.io/mgatool | bash -s v${MGA_VERSION}
	@mv bin/mga $@

bin/mockery: bin/gobin
	@mkdir -p bin
	GOBIN=bin/ bin/gobin github.com/vektra/mockery/cmd/mockery

.PHONY: generate
generate: bin/mga bin/mockery ## Generate code
	go generate -x ./...

.PHONY: validate-openapi
validate-openapi: ## Validate the openapi description
	docker run --rm -v $${PWD}:/local openapitools/openapi-generator-cli:${OPENAPI_GENERATOR_VERSION} validate --recommend -i /local/${OPENAPI_DESCRIPTOR}

.PHONY: generate-openapi
generate-openapi: validate-openapi ## Generate go server based on openapi description
	@ if [[ "$$OSTYPE" == "linux-gnu" ]]; then sudo rm -rf ./.gen/pipeline; else rm -rf ./.gen/pipeline/; fi
	docker run --rm -v $${PWD}:/local openapitools/openapi-generator-cli:${OPENAPI_GENERATOR_VERSION} generate \
	--additional-properties packageName=pipeline \
	--additional-properties withGoCodegenComment=true \
	-i /local/${OPENAPI_DESCRIPTOR} \
	-g go-server \
	-o /local/.gen/pipeline
	@ if [[ "$$OSTYPE" == "linux-gnu" ]]; then sudo chown -R $(shell id -u):$(shell id -g) .gen/pipeline/; fi
	rm .gen/pipeline/Dockerfile .gen/pipeline/README.md .gen/pipeline/main.go .gen/pipeline/go/api_* .gen/pipeline/go/logger.go .gen/pipeline/go/routers.go
	mv .gen/pipeline/go .gen/pipeline/pipeline

define generate_openapi_client
	@ if [[ "$$OSTYPE" == "linux-gnu" ]]; then sudo rm -rf ${3}; else rm -rf ${3}; fi
	docker run --rm -v $${PWD}:/local openapitools/openapi-generator-cli:${OPENAPI_GENERATOR_VERSION} generate \
	--additional-properties packageName=${2} \
	--additional-properties withGoCodegenComment=true \
	-i /local/${1} \
	-g go \
	-o /local/${3}
	@ if [[ "$$OSTYPE" == "linux-gnu" ]]; then sudo chown -R $(shell id -u):$(shell id -g) ${3}; fi
	rm ${3}/{.travis.yml,git_push.sh,go.*}
endef

apis/cloudinfo/openapi.yaml:
	@mkdir -p apis/cloudinfo
	curl https://raw.githubusercontent.com/banzaicloud/cloudinfo/${CLOUDINFO_VERSION}/api/openapi-spec/cloudinfo.yaml | sed "s/version: .*/version: ${CLOUDINFO_VERSION}/" > apis/cloudinfo/openapi.yaml

.PHONY: generate-cloudinfo-client
generate-cloudinfo-client: apis/cloudinfo/openapi.yaml ## Generate client from Cloudinfo OpenAPI spec
	$(call generate_openapi_client,apis/cloudinfo/openapi.yaml,cloudinfo,.gen/cloudinfo)

apis/anchore/swagger.yaml:
	curl https://raw.githubusercontent.com/anchore/anchore-engine/${ANCHORE_VERSION}/anchore_engine/services/apiext/swagger/swagger.yaml | tr '\n' '\r' | sed $$'s/- Images\r      - Vulnerabilities/- Images/g' | tr '\r' '\n' | sed '/- Image Content/d; /- Policy Evaluation/d; /- Queries/d' > apis/anchore/swagger.yaml

.PHONY: generate-anchore-client
generate-anchore-client: apis/anchore/swagger.yaml ## Generate client from Anchore OpenAPI spec
	$(call generate_openapi_client,apis/anchore/swagger.yaml,anchore,.gen/anchore)

bin/protoc-gen-go: bin/protoc-gen-go-${PROTOC_GEN_GO_VERSION}
	@ln -sf protoc-gen-go-${PROTOC_GEN_GO_VERSION} bin/protoc-gen-go
bin/protoc-gen-go-${PROTOC_GEN_GO_VERSION}: bin/gobin
	@mkdir -p bin
	GOBIN=bin/ bin/gobin github.com/golang/protobuf/protoc-gen-go@v${PROTOC_GEN_GO_VERSION}
	@mv bin/protoc-gen-go bin/protoc-gen-go-${PROTOC_GEN_GO_VERSION}

bin/prototool: bin/prototool-${PROTOTOOL_VERSION}
	@ln -sf prototool-${PROTOTOOL_VERSION} bin/prototool
bin/prototool-${PROTOTOOL_VERSION}:
	@mkdir -p bin
	curl -L https://github.com/uber/prototool/releases/download/v${PROTOTOOL_VERSION}/prototool-${OS}-x86_64 > ./bin/prototool-${PROTOTOOL_VERSION} && chmod +x ./bin/prototool-${PROTOTOOL_VERSION}

apis/dex/api.proto:
	@mkdir -p apis/dex
	curl https://raw.githubusercontent.com/dexidp/dex/v${DEX_VERSION}/api/api.proto > apis/dex/api.proto

.PHONY: _download-protos
_download-protos: apis/dex/api.proto

.PHONY: validate-proto
validate-proto: bin/prototool bin/protoc-gen-go _download-protos ## Validate protobuf definition
	bin/prototool $(if ${VERBOSE},--debug ,)compile
	bin/prototool $(if ${VERBOSE},--debug ,)lint
	bin/prototool $(if ${VERBOSE},--debug ,)break check

.PHONY: proto
proto: bin/prototool bin/protoc-gen-go _download-protos ## Generate client and server stubs from the protobuf definition
	bin/prototool $(if ${VERBOSE},--debug ,)all

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
