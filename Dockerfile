ARG GO_VERSION=1.12
ARG FROM_IMAGE=scratch

FROM golang:${GO_VERSION}-alpine AS builder

# set up nsswitch.conf for Go's "netgo" implementation
# https://github.com/gliderlabs/docker-alpine/issues/367#issuecomment-424546457
RUN echo 'hosts: files dns' > /etc/nsswitch.conf.build

RUN apk add --update --no-cache bash ca-certificates make curl git mercurial bzr tzdata

RUN go get -d github.com/kubernetes-sigs/aws-iam-authenticator/cmd/aws-iam-authenticator
RUN cd $GOPATH/src/github.com/kubernetes-sigs/aws-iam-authenticator && \
    git checkout 981ecbe && \
    go install ./cmd/aws-iam-authenticator

ENV GOFLAGS="-mod=readonly"
ARG GOPROXY

RUN mkdir -p /build
WORKDIR /build

COPY go.* /build/
RUN go mod download

COPY . /build
RUN make build-release

FROM ${FROM_IMAGE}

COPY --from=builder /etc/nsswitch.conf.build /etc/nsswitch.conf
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /go/bin/aws-iam-authenticator /usr/bin/
COPY --from=builder /build/views /views/
COPY --from=builder /build/templates /templates/
COPY --from=builder /build/build/release/pipeline /
COPY --from=builder /build/build/release/worker /
COPY --from=builder /build/build/release/pipelinectl /

CMD ["/pipeline"]
