FROM alpine
RUN apk add --no-cache ca-certificates
ADD pipeline /pipeline
ENTRYPOINT ["/pipeline"]
