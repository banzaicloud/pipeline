ARG GO_VERSION=1.12

FROM golang:${GO_VERSION}-alpine AS builder

RUN apk add --update --no-cache ca-certificates git

RUN go get -d github.com/kubernetes-sigs/aws-iam-authenticator/cmd/aws-iam-authenticator
RUN cd $GOPATH/src/github.com/kubernetes-sigs/aws-iam-authenticator && \
    git checkout 981ecbe && \
    go install ./cmd/aws-iam-authenticator

RUN go get github.com/derekparker/delve/cmd/dlv


FROM alpine:3.9

RUN apk add --update --no-cache ca-certificates tzdata bash curl libc6-compat

SHELL ["/bin/bash", "-c"]

COPY --from=builder /go/bin/aws-iam-authenticator /usr/bin/
COPY --from=builder /go/bin/dlv /

COPY build/debug/pipeline /
COPY build/debug/worker /
COPY build/debug/pipelinectl /
COPY views /views/
COPY templates /templates

ENTRYPOINT ["/dlv", "--listen=:40000", "--headless=true", "--api-version=2", "--log", "exec", "/pipeline-debug"]
