.DEFAULT_GOAL := help
.PHONY: help build

OS := $(shell uname -s)
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*" -not -path "./client/*")
SYMLINKS=$(shell find -L ./vendor -type l)

PKGS=$(shell go list ./... | grep -v /client)

VERSION = 0.1.0
GITREV = $(shell git rev-parse --short HEAD)

build: ## Builds binary package
	go build  -ldflags "-X main.Version=$(VERSION) -X main.GitRev=$(GITREV)" .

build-ci:
	CGO_ENABLED=0 GOOS=linux go build .

clean:
	rm -f pipeline

local: ## Starts local MySql and admin in docker
	[ -e conf/config.toml ] || cp conf/config.toml.example conf/config.toml
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

fmt:
	@gofmt -w ${GOFILES_NOVENDOR}

check-fmt:
	PKGS="${GOFILES_NOVENDOR}" GOFMT="gofmt" ./scripts/fmt-check.sh

check-misspell: install-misspell
	PKGS="${GOFILES_NOVENDOR}" MISSPELL="misspell" ./scripts/misspell-check.sh

misspell: install-misspell
	misspell -w ${GOFILES_NOVENDOR}

vet:
	@go vet -composites=false ./...

test:
	go list ./... | xargs -n1 go test -v -parallel 1 2>&1 | tee test.txt

lint: install-golint
	golint -min_confidence 0.9 -set_exit_status $(PKGS)

install-golint:
	GOLINT_CMD=$(shell command -v golint 2> /dev/null)
ifndef GOLINT_CMD
	go get github.com/golang/lint/golint
endif

install-misspell:
	MISSPELL_CMD=$(shell command -v misspell 2> /dev/null)
ifndef MISSPELL_CMD
	go get -u github.com/client9/misspell/cmd/misspell
endif

clean-vendor:
	find -L ./vendor -type l | xargs rm -rf

generate-client:
	docker run --rm -v ${PWD}:/local openapitools/openapi-generator-cli generate \
	--additional-properties packageName=client \
	-i /local/docs/openapi/pipeline.yaml \
	-g go \
	-o /local/client
	go fmt ./client

ineffassign: install-ineffassign
	ineffassign ${GOFILES_NOVENDOR}

gocyclo: install-gocyclo
	gocyclo -over 15 ${GOFILES_NOVENDOR}

install-ineffassign:
	INEFFASSIGN_CMD=$(shell command -v ineffassign 2> /dev/null)
ifndef INEFFASSIGN_CMD
	go get -u github.com/gordonklaus/ineffassign
endif

install-gocyclo:
	GOCYCLO_CMD=$(shell command -v gocyclo 2> /dev/null)
ifndef GOCYCLO_CMD
	go get -u github.com/fzipp/gocyclo
endif

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
