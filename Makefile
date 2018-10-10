SHELL=/bin/bash
OS := $(shell uname -s)
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./client/*")

VERSION ?= $(shell git rev-parse --abbrev-ref HEAD)
GITREV = $(shell git rev-parse --short HEAD 2>/dev/null)
BUILD_DATE = $(shell date +%FT%T%z)
GOLANG_VERSION = 1.11
GOLANG_VERSION_REGEX = 1.11(.[0-9]+)?

DEP_VERSION = 0.5.0
GOLANGCI_VERSION = 1.10.2
MISSPELL_VERSION = 0.3.4
JQ_VERSION = 1.5
LICENSEI_VERSION = 0.0.7
OPENAPI_GENERATOR_VERSION = v3.2.3

bin/dep: bin/dep-${DEP_VERSION}
bin/dep-${DEP_VERSION}:
	@mkdir -p bin
	@rm -rf bin/dep-*
	curl https://raw.githubusercontent.com/golang/dep/master/install.sh | INSTALL_DIRECTORY=./bin DEP_RELEASE_TAG=v${DEP_VERSION} sh
	@touch $@

.PHONY: vendor
vendor: bin/dep ## Install dependencies
	bin/dep ensure -vendor-only -v

.PHONY: build
build: ## Builds binary package
	@go version | grep -q -E "go${GOLANG_VERSION_REGEX} " || (echo "Required Go version is ${GOLANG_VERSION}\nInstalled: `go version`" && exit 1)
	go build -v -ldflags "-X main.Version=${VERSION} -X main.GitRev=${GITREV} -X main.BuildDate=${BUILD_DATE}" .

.PHONY: build-ci
build-ci:
	CGO_ENABLED=0 GOOS=linux go build -v -ldflags "-X main.Version=${VERSION} -X main.GitRev=${GITREV} -X main.BuildDate=${BUILD_DATE}" .

.PHONY: docker-build
docker-build: ## Builds go binary in docker image
	docker run -it -v $(PWD):/go/src/github.com/banzaicloud/pipeline -w /go/src/github.com/banzaicloud/pipeline golang:${GOLANG_VERSION}-alpine go build -o pipeline_linux .

.PHONY: clean
clean:
	rm -f pipeline

config/config.toml:
	cp config/config.toml.example config/config.toml

.PHONY: local
local: config/config.toml ## Starts local development environment in docker
	docker-compose -f docker-compose.yml up -d

.PHONY: local-kill
local-kill: ## Kills local development environment
	docker-compose -f docker-compose.yml kill

bin/jq: bin/jq-${JQ_VERSION}
bin/jq-${JQ_VERSION}:
	@mkdir -p bin
	@rm -rf bin/jq-*
ifeq ($(OS), Darwin)
	curl -L https://github.com/stedolan/jq/releases/download/jq-${JQ_VERSION}/jq-osx-amd64 > ./bin/jq && chmod +x ./bin/jq
endif
ifeq ($(OS), Linux)
	curl -L https://github.com/stedolan/jq/releases/download/jq-${JQ_VERSION}/jq-linux64 > ./bin/jq && chmod +x ./bin/jq
endif
	@touch $@

.PHONY: create-cluster
create-cluster: ## Curl call to pipeline api to create a cluster with your username
	curl -i -X POST http://localhost:9090/api/v1/clusters -H "Accept: application/json" -H "Content-Type: application/json" -d '{"name":"test-$(USER)","location":"eu-west-1","node":{"instanceType":"m4.large","spotPrice":"0.2","minCount":2,"maxCount":4,"image":"ami-34b6764d"},"master":{"instanceType":"m4.large","image":"ami-34b6764d"}}'

.PHONY: delete-cluster
delete-cluster: bin/jq ## Curl call to pipeline api to delete a cluster with your username
	curl -X DELETE http://localhost:9090/api/v1/clusters/$(shell curl -s localhost:9090/api/v1/clusters|bin/jq '.data[]|select(.name=="test-$(USER)")|.ID')

.PHONY: ec2-list-instances
ec2-list-instances: ## Lists aws ec2 instances, for alternative regions use: AWS_DEFAULT_REGION=us-west-2 make ec2-list-instances
	aws ec2 describe-instances --query 'Reservations[].Instances[].{ip:PublicIpAddress,id:InstanceId,state:State.Name,name:Tags[?Key==`Name`].Value|[0]}' --filters "Name=instance-state-name,Values=pending,running,shutting-down,stopping,stopped" --out table

.PHONY: generate-client
generate-client: ## Generate go client based on openapi description
	docker run --rm -v ${PWD}:/local openapitools/openapi-generator-cli:${OPENAPI_GENERATOR_VERSION} generate \
	--additional-properties packageName=client \
	--additional-properties withGoCodegenComment=true \
	-i /local/docs/openapi/pipeline.yaml \
	-g go \
	-o /local/client
	go fmt ./client

bin/golangci-lint: bin/golangci-lint-${GOLANGCI_VERSION}
bin/golangci-lint-${GOLANGCI_VERSION}:
	@mkdir -p bin
	@rm -rf bin/golangci-lint-*
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | bash -s -- -b ./bin/ v${GOLANGCI_VERSION}
	@touch $@

.PHONY: lint
lint: bin/golangci-lint ## Run linter
	@bin/golangci-lint run

.PHONY: fmt
fmt:
	@gofmt -w ${GOFILES_NOVENDOR}

bin/misspell: bin/misspell-${MISSPELL_VERSION}
bin/misspell-${MISSPELL_VERSION}:
	@mkdir -p bin
	@rm -rf bin/misspell-*
	curl -sfL https://git.io/misspell | bash -s -- -b ./bin/ v${MISSPELL_VERSION}
	@touch $@

.PHONY: misspell
misspell: bin/misspell ## Fix spelling mistakes
	misspell -w ${GOFILES_NOVENDOR}

bin/licensei: bin/licensei-${LICENSEI_VERSION}
bin/licensei-${LICENSEI_VERSION}:
	@mkdir -p bin
	@rm -rf bin/licensei-*
	curl -sfL https://raw.githubusercontent.com/goph/licensei/master/install.sh | bash -s v${LICENSEI_VERSION}
	@touch $@

.PHONY: license-check
license-check: bin/licensei ## Run license check
	bin/licensei check
	./scripts/check-header.sh

.PHONY: license-cache
license-cache: bin/licensei ## Generate license cache
	bin/licensei cache

.PHONY: test
test:
	set -o pipefail; go list ./... | xargs -n1 go test -v -parallel 1 2>&1 | tee test.txt

bin/go-junit-report: # Install JUnit report generator
	GOBIN=${PWD}/bin/ go get -u github.com/jstemmer/go-junit-report

.PHONY: junit-report
junit-report: bin/go-junit-report # Generate test reports
	@mkdir -p build
	cat test.txt | bin/go-junit-report > build/report.xml

.PHONY: list
list:
	@$(MAKE) -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort

.PHONY: help
.DEFAULT_GOAL := help
help:
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
