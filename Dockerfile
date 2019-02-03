FROM golang:1.11-alpine

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

RUN BUILD_DIR=/ make build


FROM alpine:3.7

RUN apk add --update --no-cache tzdata

COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=0 /go/bin/aws-iam-authenticator /usr/bin/
COPY --from=0 /go/src/github.com/banzaicloud/pipeline/views /views/
COPY --from=0 /go/src/github.com/banzaicloud/pipeline/templates/eks /templates/eks
COPY --from=0 /pipeline /

ENTRYPOINT ["/pipeline"]
