FROM alpine
RUN apk add --no-cache ca-certificates
ADD pipeline_static /pipeline
ENTRYPOINT ["/pipeline"]
