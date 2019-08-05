ARG GO_VERSION=1.12
ARG FROM_IMAGE=scratch

FROM golang:${GO_VERSION}-alpine AS builder

# set up nsswitch.conf for Go's "netgo" implementation
# https://github.com/gliderlabs/docker-alpine/issues/367#issuecomment-424546457
RUN echo 'hosts: files dns' > /etc/nsswitch.conf.build

RUN apk add --update --no-cache bash ca-certificates make curl git mercurial bzr tzdata

ENV GOFLAGS="-mod=readonly"
ARG GOPROXY

RUN mkdir -p /build
WORKDIR /build

COPY go.* /build/
RUN go mod download

COPY . /build
RUN make build-release


FROM alpine:3.10 AS iamauth

WORKDIR /tmp

ENV IAM_AUTH_VERSION 0.4.0
ENV IAM_AUTH_URL "https://github.com/kubernetes-sigs/aws-iam-authenticator/releases/download/v${IAM_AUTH_VERSION}"
RUN set -xe \
    && wget ${IAM_AUTH_URL}/aws-iam-authenticator_${IAM_AUTH_VERSION}_linux_amd64 \
    && wget ${IAM_AUTH_URL}/authenticator_${IAM_AUTH_VERSION}_checksums.txt \
    && cat authenticator_${IAM_AUTH_VERSION}_checksums.txt | grep "_linux_amd64" | sha256sum -c - \
    && chmod +x aws-iam-authenticator_${IAM_AUTH_VERSION}_linux_amd64 \
    && mv aws-iam-authenticator_${IAM_AUTH_VERSION}_linux_amd64 aws-iam-authenticator


FROM ${FROM_IMAGE}

COPY --from=builder /etc/nsswitch.conf.build /etc/nsswitch.conf
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=iamauth /tmp/aws-iam-authenticator /usr/bin/
COPY --from=builder /build/views /views/
COPY --from=builder /build/templates /templates/
COPY --from=builder /build/build/release/pipeline /
COPY --from=builder /build/build/release/worker /
COPY --from=builder /build/build/release/pipelinectl /

CMD ["/pipeline"]
