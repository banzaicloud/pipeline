FROM golang:1.10-alpine
RUN apk add --update --no-cache ca-certificates
ADD . /go/src/github.com/banzaicloud/pipeline
WORKDIR /go/src/github.com/banzaicloud/pipeline
RUN go build -v -o /pipeline main.go

FROM alpine:3.7
COPY --from=0 /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=0 /go/src/github.com/banzaicloud/pipeline/views /views/
COPY --from=0 /pipeline /
ENTRYPOINT ["/pipeline"]
