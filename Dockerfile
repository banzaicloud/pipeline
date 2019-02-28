FROM golang:1.11-alpine AS builder

RUN apk add --update --no-cache bash ca-certificates make curl git mercurial bzr

RUN go get -d github.com/kubernetes-sigs/aws-iam-authenticator/cmd/aws-iam-authenticator
RUN cd $GOPATH/src/github.com/kubernetes-sigs/aws-iam-authenticator && \
    git checkout 981ecbe && \
    go install ./cmd/aws-iam-authenticator

ENV GOFLAGS="-mod=readonly"

RUN mkdir -p /build
WORKDIR /build

COPY go.* /build/
RUN go mod download

COPY . /build
RUN make build-release


FROM alpine:3.9

RUN apk add --update --no-cache ca-certificates tzdata

COPY --from=builder /go/bin/aws-iam-authenticator /usr/bin/
COPY --from=builder /build/views /views/
COPY --from=builder /build/templates /templates/
COPY --from=builder /build/build/release/pipeline /
COPY --from=builder /build/build/release/worker /
COPY --from=builder /build/build/release/pipelinectl /

CMD ["/pipeline"]
