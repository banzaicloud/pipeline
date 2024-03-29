ARG GO_VERSION=1.16
ARG FROM_IMAGE=scratch

FROM golang:${GO_VERSION}-alpine3.12 AS builder

# set up nsswitch.conf for Go's "netgo" implementation
# https://github.com/gliderlabs/docker-alpine/issues/367#issuecomment-424546457
RUN echo 'hosts: files dns' > /etc/nsswitch.conf.build

RUN apk add --update --no-cache bash ca-certificates make curl git mercurial tzdata

ENV GOFLAGS="-mod=readonly"
ARG GOPROXY

RUN mkdir -p /build
WORKDIR /build

COPY go.* /build/
COPY pkg/sdk/go.* /build/pkg/sdk/
RUN go mod download

ARG VERSION
ARG COMMIT_HASH
ARG BUILD_DATE

COPY . /build
RUN make build-release


FROM alpine:3.13 AS iamauth

WORKDIR /tmp

ENV IAM_AUTH_VERSION 0.5.9
ENV IAM_AUTH_URL "https://github.com/kubernetes-sigs/aws-iam-authenticator/releases/download/v${IAM_AUTH_VERSION}"
RUN set -xe \
    && wget ${IAM_AUTH_URL}/aws-iam-authenticator_${IAM_AUTH_VERSION}_linux_amd64 \
    && wget ${IAM_AUTH_URL}/authenticator_${IAM_AUTH_VERSION}_checksums.txt \
    && cat authenticator_${IAM_AUTH_VERSION}_checksums.txt | grep "_linux_amd64" | sha256sum -c - \
    && chmod +x aws-iam-authenticator_${IAM_AUTH_VERSION}_linux_amd64 \
    && mv aws-iam-authenticator_${IAM_AUTH_VERSION}_linux_amd64 aws-iam-authenticator

FROM alpine:3.13 AS helms3

WORKDIR /tmp

ENV HELM_S3_PLUGIN_VERSION=0.11.0

ENV HELM_S3_ARCHIVE_FILE_NAME="helm-s3_${HELM_S3_PLUGIN_VERSION}_linux_amd64.tar.gz" \
    HELM_S3_CHECKSUMS_FILE_NAME="helm-s3_${HELM_S3_PLUGIN_VERSION}_sha512_checksums.txt" \
    HELM_S3_PLUGIN_URL="https://github.com/banzaicloud/helm-s3/releases/download/v${HELM_S3_PLUGIN_VERSION}"

RUN set -xe \
    && mkdir -p helm-plugins/helm-s3 \
    && wget "${HELM_S3_PLUGIN_URL}/${HELM_S3_ARCHIVE_FILE_NAME}" \
    && wget "${HELM_S3_PLUGIN_URL}/${HELM_S3_CHECKSUMS_FILE_NAME}" \
    && cat "${HELM_S3_CHECKSUMS_FILE_NAME}" | grep -E "^[0-9a-z]+  ${HELM_S3_ARCHIVE_FILE_NAME}$" | sha512sum -c - \
    && tar xzf "${HELM_S3_ARCHIVE_FILE_NAME}" -C "helm-plugins/helm-s3"

FROM alpine:3.13 AS migrate

ENV MIGRATE_VERSION v4.9.1

RUN set -xe && \
    apk add --update --no-cache ca-certificates curl && \
    curl -L https://github.com/golang-migrate/migrate/releases/download/${MIGRATE_VERSION}/migrate.linux-amd64.tar.gz | tar xvz && \
    mv migrate.linux-amd64 /tmp/migrate


FROM ${FROM_IMAGE}

ENV HELM_PLUGINS=/usr/share/helm/plugins

COPY --from=builder /etc/nsswitch.conf.build /etc/nsswitch.conf
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=iamauth /tmp/aws-iam-authenticator /usr/bin/
COPY --from=helms3 /tmp/helm-plugins /usr/share/helm/plugins
COPY --from=migrate /tmp/migrate /usr/bin/
COPY --from=builder /build/database/migrations /migrations/
COPY --from=builder /build/templates /templates/
COPY --from=builder /build/build/release/pipeline /
COPY --from=builder /build/build/release/worker /
COPY --from=builder /build/build/release/pipelinectl /
COPY config/anchore/policies/ /policies/

CMD ["/pipeline"]
