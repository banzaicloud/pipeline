FROM golang:1.11-alpine AS builder

RUN apk add --update --no-cache bash ca-certificates curl git make

RUN go get -d github.com/kubernetes-sigs/aws-iam-authenticator/cmd/aws-iam-authenticator
RUN cd $GOPATH/src/github.com/kubernetes-sigs/aws-iam-authenticator && \
    git checkout 981ecbe && \
    go install ./cmd/aws-iam-authenticator

RUN mkdir -p /go/src/github.com/banzaicloud/pipeline
ADD Gopkg.* Makefile /go/src/github.com/banzaicloud/pipeline/

WORKDIR /go/src/github.com/banzaicloud/pipeline

RUN make vendor

ADD . /go/src/github.com/banzaicloud/pipeline

RUN BUILD_DIR=/build make build-release


FROM alpine:3.8

RUN apk add --update --no-cache ca-certificates tzdata

COPY --from=builder /go/bin/aws-iam-authenticator /usr/bin/
COPY --from=builder /go/src/github.com/banzaicloud/pipeline/views /views/
COPY --from=builder /go/src/github.com/banzaicloud/pipeline/templates/eks /templates/eks/
COPY --from=builder /build/release/pipeline /
COPY --from=builder /build/release/worker /

CMD ["/pipeline"]
