ARG GO_VERSION=1.12

FROM golang:${GO_VERSION}-alpine AS builder

RUN apk add --update --no-cache ca-certificates git

RUN go get -d github.com/kubernetes-sigs/aws-iam-authenticator/cmd/aws-iam-authenticator
RUN cd $GOPATH/src/github.com/kubernetes-sigs/aws-iam-authenticator && \
    git checkout 981ecbe && \
    go install ./cmd/aws-iam-authenticator


FROM alpine:3.9

RUN apk add --update --no-cache ca-certificates tzdata bash curl

SHELL ["/bin/bash", "-c"]

COPY --from=builder /go/bin/aws-iam-authenticator /usr/bin/

COPY build/release/pipeline /
COPY build/release/worker /
COPY build/release/pipelinectl /
COPY views /views/
COPY templates/ /templates/

CMD ["/pipeline"]
