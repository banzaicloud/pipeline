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
TEMPORARY_DIRECTORY = tmp
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || git symbolic-ref -q --short HEAD)
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null)
BUILD_DATE ?= $(shell date +%FT%T%z)
HELM_VERSION = $(shell cat go.mod | grep helm.sh/helm/v3 | grep -v "=>" | cut -d" " -f2 | sed s/^v//)
LDFLAGS += -X main.version=${VERSION} -X main.commitHash=${COMMIT_HASH} -X main.buildDate=${BUILD_DATE} -X main.helmVersion=${HELM_VERSION}
export CGO_ENABLED ?= 0
ifeq ($(VERBOSE), 1)
ifeq ($(filter -v,${GOARGS}),)
	GOARGS += -v
endif
TEST_FORMAT = short-verbose
endif

CLOUDINFO_VERSION = 0.9.5
DEX_VERSION = 2.19.0
# TODO: use an exact version
ANCHORE_VERSION = 156836d

GOLANGCI_VERSION = 1.30.0
LICENSEI_VERSION = 0.3.1
OPENAPI_GENERATOR_VERSION = v4.3.1
MIGRATE_VERSION = 4.9.1
GOTESTSUM_VERSION = 0.4.1
MGA_VERSION = 0.4.2
GRYPE_VERSION = 0.11.0

GOLANG_VERSION = 1.16

export PIPELINE_CONFIG_DIR ?= $(PWD)/config

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
	# Waiting for Dex to initialize with potential container restarts.
	@ while ! docker-compose logs dex | grep -q "listening (http)" ; do sleep 1 ; echo "." ; done

.PHONY: stop
stop: ## Stop docker development environment
	docker-compose stop

config/config.yaml:
	cat config/config.dev.yaml | sed "s/uuid: \"\"/uuid: \"$$RANDOM.$$USER.local\"/" > config/config.yaml

config/ui/feature-set.json:
	mv config/ui/feature-set.json{,~} || true && cp config/ui/feature-set.json.dist config/ui/feature-set.json

etc/config/dex.yml:
	cp etc/config/dex.yml.dist etc/config/dex.yml

.PHONY: run
run: GOTAGS += dev
run: build-pipeline ## Build and execute a binary
	PIPELINE_CONFIG_DIR=$(PIPELINE_CONFIG_DIR) VAULT_ADDR="http://127.0.0.1:8200" ${BUILD_DIR}/${BINARY_NAME} ${ARGS}

.PHONY: debug
debug: GOTAGS += dev
debug: builddebug-pipeline
	PIPELINE_CONFIG_DIR=$(PIPELINE_CONFIG_DIR) VAULT_ADDR="http://127.0.0.1:8200" dlv --listen=:40000 --log --headless=true --api-version=2 exec build/debug/pipeline -- $(ARGS)

.PHONY: debug-worker
debug-worker: GOTAGS += dev
debug-worker: builddebug-worker
	PIPELINE_CONFIG_DIR=$(PIPELINE_CONFIG_DIR) VAULT_ADDR="http://127.0.0.1:8200" dlv --listen=:40000 --log --headless=true --api-version=2 exec build/debug/worker -- $(ARGS)

.PHONY: run-worker
run-worker: GOTAGS += dev
run-worker: build-worker ## Build and execute a binary
	PIPELINE_CONFIG_DIR=$(PIPELINE_CONFIG_DIR) VAULT_ADDR="http://127.0.0.1:8200" ${BUILD_DIR}/worker ${ARGS}

.PHONY: runall ## Run worker and pipeline in foreground. Use with make -j.
runall: run run-worker

.PHONY: goversion
goversion:
ifneq ($(IGNORE_GOLANG_VERSION_REQ), 1)
	@printf "${GOLANG_VERSION}\n$$(go version | awk '{sub(/^go/, "", $$3);print $$3}')" | sort -t '.' -k 1,1 -k 2,2 -k 3,3 -g | head -1 | grep -q -E "^${GOLANG_VERSION}$$" || (printf "Required Go version is ${GOLANG_VERSION}\nInstalled: `go version`" && exit 1)
endif

.PHONY: build-%
build-%: goversion ## Build a binary
	go build ${GOARGS} -tags "${GOTAGS}" -ldflags "${LDFLAGS}" -o ${BUILD_DIR}/$* ./cmd/$*

.PHONY: builddebug-%
builddebug-%: goversion ## Build a binary
	@${MAKE} GOARGS="${GOARGS} -gcflags \"all=-N -l\"" BUILD_DIR="${BUILD_DIR}/debug" build-$*

.PHONY: build
build: goversion ## Build all binaries
ifeq ($(VERBOSE), 1)
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
	cd pkg/sdk && ../../bin/golangci-lint run

.PHONY: fix
fix: export CGO_ENABLED = 1
fix: bin/golangci-lint ## Fix lint violations
	bin/golangci-lint run --fix
	cd pkg/sdk && ../../bin/golangci-lint run --fix

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
	cd pkg/sdk && ../../bin/gotestsum --no-summary=skipped --junitfile ../../${BUILD_DIR}/test_results/${TEST_REPORT}/${TEST_REPORT_NAME} --format ${TEST_FORMAT} -- $(filter-out -v,${GOARGS}) $(if ${TEST_PKGS},${TEST_PKGS},./...)

.PHONY: test-all
test-all: ## Run all tests
	@${MAKE} GOARGS="${GOARGS} -run .\*" TEST_REPORT=all test

.PHONY: test-integration
test-integration: bin/test/kube-apiserver bin/test/etcd ## Run integration tests
	@${MAKE} TEST_ASSET_KUBE_APISERVER=$(abspath bin/test/kube-apiserver) TEST_ASSET_ETCD=$(abspath bin/test/etcd) GOARGS="${GOARGS} -run ^TestIntegration\$$\$$" TEST_REPORT=integration test

.PHONY: test-integrated-service-up
test-integrated-service-up: ## Run integrated service functional tests
	@echo "Stopping pipeline development stack if it's already running"
	docker-compose stop
	if ! kind get kubeconfig --name pipeline-is-test 1>/dev/null; then kind create cluster --name pipeline-is-test --kubeconfig $(HOME)/.kube/kind-pipeline-is-test; fi
	mkdir -p .docker/volumes/{mysql,vault/file,vault/keys}
	uid=$(shell id -u) gid=$(shell id -g) docker-compose -p pipeline-is-test --project-directory $(PWD) -f $(PWD)/internal/integratedservices/testconfig/docker-compose.yml up -d
	@while ! test -f $(HOME)/.vault-token; do echo "waiting for vault root token"; docker ps; docker-compose logs --tail 10; sleep 3; done
	ls -alh $(HOME)/.vault-token

IS_TEST_ENTRYPOINT = TestV1

.PHONY: test-integrated-service
test-integrated-service: ## Run integrated service functional tests
	cd internal/integratedservices; \
		PIPELINE_CONFIG_DIR=$(PWD)/internal/integratedservices/testconfig \
		KUBECONFIG=<(kind get kubeconfig --name pipeline-is-test) \
		VAULT_ADDR="http://127.0.0.1:8200" \
		go test -v -run ^$(IS_TEST_ENTRYPOINT)

.PHONY: test-integrated-service-v2
test-integrated-service-v2: IS_TEST_ENTRYPOINT = TestV2 ## Run integrated service functional tests for v2
test-integrated-service-v2: test-integrated-service

IS_WORKER_CONFIG = $(PWD)/internal/integratedservices/testconfig/config.yaml

.PHONY: test-integrated-service-worker
test-integrated-service-worker: GOTAGS += dev
test-integrated-service-worker: build-worker
	VAULT_ADDR="http://127.0.0.1:8200" ${BUILD_DIR}/worker --config $(IS_WORKER_CONFIG) ${ARGS}

.PHONY: test-integrated-service-worker-v2
test-integrated-service-worker-v2: IS_WORKER_CONFIG = $(PWD)/internal/integratedservices/testconfig/config-v2.yaml
test-integrated-service-worker-v2: test-integrated-service-worker

.PHONY: test-integrated-service-down
test-integrated-service-down:
	docker-compose -p pipeline-is-test --project-directory $(PWD) -f $(PWD)/internal/integratedservices/testconfig/docker-compose.yml down
	kind delete clusters pipeline-is-test

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

bin/mga: bin/mga-${MGA_VERSION}
	@ln -sf mga-${MGA_VERSION} bin/mga
bin/mga-${MGA_VERSION}:
	@mkdir -p bin
	curl -sfL https://git.io/mgatool | bash -s v${MGA_VERSION}
	@mv bin/mga $@

.PHONY: generate-all
generate-all: generate-anchore-client generate-cloudinfo-client generate generate-openapi

.PHONY: generate
generate: bin/mga ## Generate code
	go generate -x ./...
	bin/mga gen kit endpoint ./...
	bin/mga gen ev dispatcher ./...
	bin/mga gen ev handler ./...
	bin/mga gen testify mock ./...

.PHONY: validate-openapi
validate-openapi: ## Validate the openapi description
	docker run --rm -v $${PWD}:/local openapitools/openapi-generator-cli:${OPENAPI_GENERATOR_VERSION} validate --recommend -i /local/${OPENAPI_DESCRIPTOR}

.PHONY: generate-openapi
generate-openapi: validate-openapi ## Generate go server based on openapi description
	$(call back_up_file,.gen/pipeline/pipeline/BUILD.plz)
	@ if [[ "$$OSTYPE" == "linux-gnu" ]]; then sudo rm -rf ./.gen/pipeline; else rm -rf ./.gen/pipeline/; fi
	docker run --rm -v $${PWD}:/local openapitools/openapi-generator-cli:${OPENAPI_GENERATOR_VERSION} generate \
	--additional-properties packageName=pipeline \
	--additional-properties withGoCodegenComment=true \
	-i /local/${OPENAPI_DESCRIPTOR} \
	-g go-server \
	-o /local/.gen/pipeline
	@ if [[ "$$OSTYPE" == "linux-gnu" ]]; then sudo chown -R $(shell id -u):$(shell id -g) .gen/pipeline/; fi
	rm .gen/pipeline/{Dockerfile,go.*,README.md,main.go,go/api*.go,go/logger.go,go/routers.go}
	mv .gen/pipeline/go .gen/pipeline/pipeline
	$(call restore_backup_file,.gen/pipeline/pipeline/BUILD.plz)

define generate_openapi_client
	@ if [[ "$$OSTYPE" == "linux-gnu" ]]; then sudo rm -rf ${3}; else rm -rf ${3}; fi
	docker run --rm -v $${PWD}:/local openapitools/openapi-generator-cli:${OPENAPI_GENERATOR_VERSION} generate \
	--additional-properties packageName=${2} \
	--additional-properties withGoCodegenComment=true \
	-i /local/${1} \
	-g go \
	-o /local/${3}
	@ if [[ "$$OSTYPE" == "linux-gnu" ]]; then sudo chown -R $(shell id -u):$(shell id -g) ${3}; fi
	rm -rf ${3}/{.travis.yml,git_push.sh,go.*,docs}
endef

apis/cloudinfo/openapi.yaml:
	@mkdir -p apis/cloudinfo
	curl https://raw.githubusercontent.com/banzaicloud/cloudinfo/${CLOUDINFO_VERSION}/api/openapi-spec/cloudinfo.yaml | sed "s/version: .*/version: ${CLOUDINFO_VERSION}/" > apis/cloudinfo/openapi.yaml

.PHONY: generate-cloudinfo-client
generate-cloudinfo-client: apis/cloudinfo/openapi.yaml ## Generate client from Cloudinfo OpenAPI spec
	$(call back_up_file,.gen/cloudinfo/BUILD.plz)
	$(call generate_openapi_client,apis/cloudinfo/openapi.yaml,cloudinfo,.gen/cloudinfo)
	$(call restore_backup_file,.gen/cloudinfo/BUILD.plz)

apis/anchore/swagger.yaml:
	curl https://raw.githubusercontent.com/anchore/anchore-engine/${ANCHORE_VERSION}/anchore_engine/services/apiext/swagger/swagger.yaml | tr '\n' '\r' | sed $$'s/- Images\r      - Vulnerabilities/- Images/g' | tr '\r' '\n' | sed '/- Image Content/d; /- Policy Evaluation/d; /- Queries/d' > apis/anchore/swagger.yaml

.PHONY: generate-anchore-client
generate-anchore-client: ## apis/anchore/swagger.yaml ## https://github.com/anchore/anchore-engine/pull/846 ## Generate client from Anchore OpenAPI spec
	$(call back_up_file,.gen/anchore/BUILD.plz)
	$(call generate_openapi_client,apis/anchore/swagger.yaml,anchore,.gen/anchore)
	@ sed -i~ 's/whitelist_ids,omitempty/whitelist_ids/' .gen/anchore/model_mapping_rule.go && rm .gen/anchore/model_mapping_rule.go~
	@ sed -i~ 's/params,omitempty/params/' .gen/anchore/model_policy_rule.go && rm .gen/anchore/model_policy_rule.go~
	$(call restore_backup_file,.gen/anchore/BUILD.plz)

bin/grype: bin/grype-${GRYPE_VERSION}
	@ln -sf grype-${GRYPE_VERSION} bin/grype
bin/grype-${GRYPE_VERSION}:
	@mkdir -p bin
	curl -sfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | bash -s -- -b ./bin/ v${GRYPE_VERSION}
	@mv bin/grype $@
scan-docker-images: bin/grype
	@echo "- Start vulnerablity scan for images: $$(cat docker.images.list)"
	@for image in $$(cat docker.images.list); do echo "Scanning image: " $$image; grype $$image; done;
	@echo "- Scan completed."

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

# back_up_file copies the specified file into the temporary directory
# $(TEMPORARY_DIRECTORY) under the path it was located relative to the
# repository root. This method has a pair and inverse operation named
# `restore_backup_file` which restores the backed up file to its original place.
# The original entity is left in place during backup.
#
# $1 - source file path to back up.
define back_up_file
	$(eval source_file_path := $(1))
	@echo "- Backing up $(source_file_path)."

	$(call check_binary,realpath,coreutils)

	$(eval source_file_path := $(shell realpath --relative-to=. $(source_file_path)))
	$(eval backup_file_path := $(shell echo "$(TEMPORARY_DIRECTORY)/$(source_file_path)"))
	$(eval backup_directory := $(shell dirname $(backup_file_path)))

	@mkdir -p $(backup_directory)
	@cp -p $(source_file_path) $(backup_file_path)

	@echo "- Backup completed."
endef

# check_binary checks for the specified binary to exist and errors if it is
# missing.
#
# $1 - binary to check.
#
# $2 - package name.
define check_binary
	$(eval binary_name := $(1))
	$(eval package_name := $(2))

	$(eval binary_path := $(shell which $(binary_name)))
	$(if $(binary_path),,$(error $(binary_name) binary not found, please install it with 'brew install $(package_name)' or a similar command regarding other package managers))
endef

# restore_backup_file restores the specified file's backup from the temporary
# directory $(TEMPORARY_DIRECTORY) to its original path by taking its relative
# path under the temporary directory and moving it into the repository root
# directory without the temporary directory fragment. This method is the pair
# and inverse operation of back_up_file. The copy entity in the temporary
# directory is then removed and all empty directories in and including the
# temporary directory are also removed.
#
# $1 - target file path whose backup is to be restored from the temporary
# directory.
define restore_backup_file
	$(eval target_file_path := $(1))
	@echo "- Restoring $(target_file_path)."

	$(call check_binary,realpath,coreutils)

	$(eval target_file_path := $(shell realpath --relative-to=. $(target_file_path)))
	$(eval backup_file_path := $(shell echo "$(TEMPORARY_DIRECTORY)/$(target_file_path)"))
	$(eval target_directory := $(shell dirname $(target_file_path)))

	@mkdir -p $(target_directory)
	@cp -p $(backup_file_path) $(target_file_path)
	@rm $(backup_file_path)
	@find $(TEMPORARY_DIRECTORY) -type d -empty -delete

	@echo "- Restore completed."
endef
