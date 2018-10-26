# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html

SHELL = /bin/bash
OS = $(shell uname -s)

# Project variables
PACKAGE = github.com/banzaicloud/pipeline
BINARY_NAME = pipeline
OPENAPI_DESCRIPTOR = docs/openapi/pipeline.yaml

# Build variables
BUILD_DIR ?= build
BUILD_PACKAGE = ${PACKAGE}/cmd/pipeline
VERSION ?= $(shell git rev-parse --abbrev-ref HEAD)
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null)
BUILD_DATE ?= $(shell date +%FT%T%z)
LDFLAGS += -X main.Version=${VERSION} -X main.CommitHash=${COMMIT_HASH} -X main.BuildDate=${BUILD_DATE}
export CGO_ENABLED ?= 0
ifeq (${VERBOSE}, 1)
	GOARGS += -v
endif

DEP_VERSION = 0.5.0
GOLANGCI_VERSION = 1.10.2
MISSPELL_VERSION = 0.3.4
JQ_VERSION = 1.5
LICENSEI_VERSION = 0.0.7
OPENAPI_GENERATOR_VERSION = 3.3.0

GOLANG_VERSION = 1.11

GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./client/*")

.PHONY: up
up: vendor start config/config.toml ## Set up the development environment

.PHONY: down
down: clean ## Destroy the development environment
	docker-compose down
	rm -rf .docker/

.PHONY: reset
reset: down up ## Reset the development environment

.PHONY: clean
clean: ## Clean the working area and the project
	rm -rf bin/ ${BUILD_DIR}/ vendor/
	rm -rf pipeline

docker-compose.override.yml: ## Create docker compose override file
	cp docker-compose.override.yml.dist docker-compose.override.yml

docker-compose.anchore.yml: ## Create docker compose override file with anchore
	cp docker-compose.anchore.yml.dist docker-compose.override.yml

.PHONY: start
start: docker-compose.override.yml ## Start docker development environment
	docker-compose up -d

.PHONY: anchorestart
anchorestart: docker-compose.anchore.yml  ## Start docker development environment with anchore
	docker-compose up -d

.PHONY: stop
stop: ## Stop docker development environment
	docker-compose stop

bin/dep: bin/dep-${DEP_VERSION}
	@ln -sf dep-${DEP_VERSION} bin/dep
bin/dep-${DEP_VERSION}:
	@mkdir -p bin
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | INSTALL_DIRECTORY=bin DEP_RELEASE_TAG=v${DEP_VERSION} sh
	@mv bin/dep $@

.PHONY: vendor
vendor: bin/dep ## Install dependencies
	bin/dep ensure -v -vendor-only

config/config.toml:
	cp config/config.toml.example config/config.toml

.PHONY: build
build: GOARGS += -tags "${GOTAGS}" -ldflags "${LDFLAGS}" -o ${BUILD_DIR}/${BINARY_NAME}
build: ## Build a binary
ifneq (${IGNORE_GOLANG_VERSION_REQ}, 1)
	@printf "${GOLANG_VERSION}\n$$(go version | awk '{sub(/^go/, "", $$3);print $$3}')" | sort -t '.' -k 1,1 -k 2,2 -k 3,3 -g | head -1 | grep -q -E "^${GOLANG_VERSION}$$" || (printf "Required Go version is ${GOLANG_VERSION}\nInstalled: `go version`" && exit 1)
endif
	go build ${GOARGS} ${BUILD_PACKAGE}

.PHONY: docker-build
docker-build: ## Builds go binary in docker image
	docker run -it -v $(PWD):/go/src/${PACKAGE} -w /go/src/${PACKAGE} golang:${GOLANG_VERSION}-alpine go build -o pipeline_linux ${BUILD_PACKAGE}

.PHONY: debug
debug: GOARGS += -gcflags "-N -l"
debug: BINARY_NAME = pipeline-debug
debug: build ## Builds binary package

.PHONY: debug-docker
debug-docker: debug ## Builds binary package
	docker build -t banzaicloud/pipeline:debug -f Dockerfile.dev .

.PHONY: check
check: test lint ## Run tests and linters

bin/golangci-lint: bin/golangci-lint-${GOLANGCI_VERSION}
	@ln -sf golangci-lint-${GOLANGCI_VERSION} bin/golangci-lint
bin/golangci-lint-${GOLANGCI_VERSION}:
	@mkdir -p bin
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | bash -s -- -b ./bin/ v${GOLANGCI_VERSION}
	@mv bin/golangci-lint $@

.PHONY: lint
lint: bin/golangci-lint ## Run linter
	@bin/golangci-lint run

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

.PHONY: test
test:
	set -o pipefail; go list ./... | xargs -n1 go test ${GOARGS} -v -parallel 1 2>&1 | tee test.txt

bin/go-junit-report:
	@mkdir -p bin
	GOBIN=${PWD}/bin/ go get -u github.com/jstemmer/go-junit-report

.PHONY: junit-report
junit-report: bin/go-junit-report # Generate test reports
	@mkdir -p build
	cat test.txt | bin/go-junit-report > build/report.xml

.PHONY: validate-openapi
validate-openapi: ## Validate the openapi description
	docker run --rm -v ${PWD}:/local openapitools/openapi-generator-cli:v${OPENAPI_GENERATOR_VERSION} validate --recommend -i /local/${OPENAPI_DESCRIPTOR}

.PHONY: generate-client
generate-client: validate-openapi ## Generate go client based on openapi description
	rm -rf ./client
	docker run --rm -v ${PWD}:/local openapitools/openapi-generator-cli:v${OPENAPI_GENERATOR_VERSION} generate \
	--additional-properties packageName=client \
	--additional-properties withGoCodegenComment=true \
	-i /local/${OPENAPI_DESCRIPTOR} \
	-g go \
	-o /local/client
	gofmt -s -w client/

ifeq (${OS}, Darwin)
	shasum -a 256 ${OPENAPI_DESCRIPTOR} > client/SHA256SUMS
endif
ifeq (${OS}, Linux)
	sha256sum ${OPENAPI_DESCRIPTOR} > client/SHA256SUMS
endif

bin/jq: bin/jq-${JQ_VERSION}
	@ln -sf jq-${JQ_VERSION} bin/jq
bin/jq-${JQ_VERSION}:
	@mkdir -p bin
ifeq (${OS}, Darwin)
	curl -L https://github.com/stedolan/jq/releases/download/jq-${JQ_VERSION}/jq-osx-amd64 > ./bin/jq-${JQ_VERSION} && chmod +x ./bin/jq-${JQ_VERSION}
endif
ifeq (${OS}, Linux)
	curl -L https://github.com/stedolan/jq/releases/download/jq-${JQ_VERSION}/jq-linux64 > ./bin/jq-${JQ_VERSION} && chmod +x ./bin/jq-${JQ_VERSION}
endif

.PHONY: create-cluster
create-cluster: ## Curl call to pipeline api to create a cluster with your username
	curl -i -X POST http://localhost:9090/api/v1/clusters -H "Accept: application/json" -H "Content-Type: application/json" -d '{"name":"test-$(USER)","location":"eu-west-1","node":{"instanceType":"m4.large","spotPrice":"0.2","minCount":2,"maxCount":4,"image":"ami-34b6764d"},"master":{"instanceType":"m4.large","image":"ami-34b6764d"}}'

.PHONY: delete-cluster
delete-cluster: bin/jq ## Curl call to pipeline api to delete a cluster with your username
	curl -X DELETE http://localhost:9090/api/v1/clusters/$(shell curl -s localhost:9090/api/v1/clusters|bin/jq '.data[]|select(.name=="test-$(USER)")|.ID')

.PHONY: ec2-list-instances
ec2-list-instances: ## Lists aws ec2 instances, for alternative regions use: AWS_DEFAULT_REGION=us-west-2 make ec2-list-instances
	aws ec2 describe-instances --query 'Reservations[].Instances[].{ip:PublicIpAddress,id:InstanceId,state:State.Name,name:Tags[?Key==`Name`].Value|[0]}' --filters "Name=instance-state-name,Values=pending,running,shutting-down,stopping,stopped" --out table

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
