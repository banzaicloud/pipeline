.DEFAULT_GOAL := help
.PHONY: help build

OS := $(shell uname -s)
GOFILES_NOVENDOR = $(shell find . -type f -name '*.go' -not -path "./vendor/*")

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
	docker-compose -f docker-compose-local.yml up -d

local-kill: ## Kills local MySql and admin
	docker-compose -f docker-compose-local.yml kill

docker-build: docker-dev-img ## Builds go binary in docker image
	docker run -it -v $(PWD):/go/src/github.com/banzaicloud/pipeline -w /go/src/github.com/banzaicloud/pipeline pipeline-primary go build -o pipeline_linux .

deps: ## Install dependencies required for building
	which glide > /dev/null || go get github.com/Masterminds/glide
	which glide-vc > /dev/null || go get github.com/sgotti/glide-vc
	which circleci  > /dev/null || curl -o /usr/local/bin/circleci https://circle-downloads.s3.amazonaws.com/releases/build_agent_wrapper/circleci && chmod +x /usr/local/bin/circleci

ifeq ($(OS), Darwin)
	which jq  > /dev/null || curl -L https://github.com/stedolan/jq/releases/download/jq-1.5/jq-osx-amd64 > /usr/local/bin/jq && chmod +x /usr/local/bin/jq
endif
ifeq ($(OS), Linux)
	which jq  > /dev/null || curl -L https://github.com/stedolan/jq/releases/download/jq-1.5/jq-linux64 > /usr/local/bin/jq && chmod +x /usr/local/bin/jq
endif

revendor: ## fix vendor dir (flattened) with forked kubicorn
	rm -rf vendor
	glide i -v --skip-test
	rm -rf vendor/github.com/kubicorn/kubicorn/
	git clone https://github.com/banzaicloud/kubicorn.git  vendor/github.com/kubicorn/kubicorn/
	cd vendor/github.com/kubicorn/kubicorn/ &&  git checkout master
	glide-vc --only-code --no-tests

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

vet:
	@go vet -composites=false ./...

test:
	@go test -v ./...
