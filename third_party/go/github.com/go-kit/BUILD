package(default_visibility = ["PUBLIC"])

go_get(
    name = "kit",
    get = "github.com/go-kit/kit",
    install = [
        "endpoint",
        "log",
        "transport",
        "transport/http",
        "sd",
        "sd/lb",
        "transport/grpc",
        "tracing/opencensus",
    ],
    sum = "h1:dXFJfIHVvUcpSgDOV+Ne6t7jXri8Tfv2uOLHUZ2XNuo=",
    version = "v0.10.0",
    deps = [
        "//third_party/go:go.opencensus.io",
        "//third_party/go/github.com/go-logfmt:logfmt",
        "//third_party/go/google.golang.org:grpc",
    ],
)
