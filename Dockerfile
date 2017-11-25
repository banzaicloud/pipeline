FROM golang:1.9-alpine

ADD . /go/src/github.com/banzaicloud/pipeline
WORKDIR /go/src/github.com/banzaicloud/pipeline
RUN go build -o /pipeline main.go

FROM alpine:3.6
RUN apk add --no-cache ca-certificates
COPY --from=0 /pipeline /
ENTRYPOINT ["/pipeline"]
