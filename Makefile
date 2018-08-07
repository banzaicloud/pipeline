.DEFAULT_GOAL := help
.PHONY: help build

OS := $(shell uname -s)
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./client/*")
SYMLINKS=$(shell find -L ./vendor -type l)

PKGS=$(shell go list ./... | grep -v /client)

VERSION = 0.1.0
GITREV = $(shell git rev-parse --short HEAD)

GOLANGCI_VERSION = 1.9.3
MISSPELL_VERSION = 0.3.4

build: ## Builds binary package
	go build -v -ldflags "-X main.Version=$(VERSION) -X main.GitRev=$(GITREV)" .

build-ci:
	CGO_ENABLED=0 GOOS=linux go build .

clean:
	rm -f pipeline

local: ## Starts local MySql and admin in docker
	[ -e config/config.toml ] || cp config/config.toml.example config/config.toml
	docker-compose -f docker-compose.yml up -d

local-kill: ## Kills local MySql and admin
	docker-compose -f docker-compose.yml kill

docker-build: ## Builds go binary in docker image
	docker run -it -v $(PWD):/go/src/github.com/banzaicloud/pipeline -w /go/src/github.com/banzaicloud/pipeline golang:1.10.1-alpine go build -o pipeline_linux .

deps: ## Install dependencies required for building
	which dep > /dev/null || brew install dep

ifeq ($(OS), Darwin)
	which jq  > /dev/null || curl -L https://github.com/stedolan/jq/releases/download/jq-1.5/jq-osx-amd64 > /usr/local/bin/jq && chmod +x /usr/local/bin/jq
endif
ifeq ($(OS), Linux)
	which jq  > /dev/null || curl -L https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 > /usr/local/bin/jq && chmod +x /usr/local/bin/jq
endif

help: ## Generates this help message
	@grep -E '^[0-9a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

create-cluster: ## Curl call to pipeline api to create a cluster with your username
	curl -i -X POST http://localhost:9090/api/v1/clusters -H "Accept: application/json" -H "Content-Type: application/json" -d '{"name":"test-$(USER)","location":"eu-west-1","node":{"instanceType":"m4.large","spotPrice":"0.2","minCount":2,"maxCount":4,"image":"ami-34b6764d"},"master":{"instanceType":"m4.large","image":"ami-34b6764d"}}'

delete-cluster: ## Curl call to pipeline api to delete a cluster with your username
	curl -X DELETE http://localhost:9090/api/v1/clusters/$(shell curl -s localhost:9090/api/v1/clusters|jq '.data[]|select(.name=="test-$(USER)")|.ID')

ec2-list-instances: ## Lists aws ec2 instances, for alternative regions use: AWS_DEFAULT_REGION=us-west-2 make ec2-list-instances
	aws ec2 describe-instances --query 'Reservations[].Instances[].{ip:PublicIpAddress,id:InstanceId,state:State.Name,name:Tags[?Key==`Name`].Value|[0]}' --filters "Name=instance-state-name,Values=pending,running,shutting-down,stopping,stopped" --out table

list:
	@$(MAKE) -pRrn : -f $(MAKEFILE_LIST) 2>/dev/null | awk -v RS= -F: '/^# File/,/^# Finished Make data base/ {if ($$1 !~ "^[#.]") {print $$1}}' | egrep -v -e '^[^[:alnum:]]' -e '^$@$$' | sort

bin/golangci-lint: ## Install golangci linter
	@mkdir -p ./bin/
	curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | bash -s -- -b ./bin/ v${GOLANGCI_VERSION}

.PHONY: lint
lint: bin/golangci-lint ## Run linter
	@bin/golangci-lint run

fmt:
	@gofmt -w ${GOFILES_NOVENDOR}

bin/misspell: ## Install misspell
	@mkdir -p ./bin/
	curl -sfL https://git.io/misspell | bash -s -- -b ./bin/ v${MISSPELL_VERSION}

misspell: bin/misspell ## Fix spelling mistakes
	misspell -w ${GOFILES_NOVENDOR}

test:
	./scripts/test.sh

clean-vendor:
	find -L ./vendor -type l | xargs rm -rf

generate-client:
	docker run --rm -v ${PWD}:/local openapitools/openapi-generator-cli generate \
	--additional-properties packageName=client \
	-i /local/docs/openapi/pipeline.yaml \
	-g go \
	-o /local/client
	go fmt ./client

check-symlinks:
	FILES="${SYMLINKS}" ./scripts/symlink-check.sh

install-go-junit-report:
	GOLINT_CMD=$(shell command -v go-junit-report 2> /dev/null)
ifndef GOLINT_CMD
	go get -u github.com/jstemmer/go-junit-report
endif

go-junit-report: install-go-junit-report
	$(shell mkdir -p test-results)
	cat test.txt | go-junit-report > test-results/report.xml
