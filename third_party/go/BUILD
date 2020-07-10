package(default_visibility = ["PUBLIC"])

go_get(
    name = "go.opencensus.io",
    get = "go.opencensus.io",
    install = [
        "",
        "internal",
        "trace/internal",
        "trace/tracestate",
        "trace",
        "trace/propagation",
        "plugin/ochttp/propagation/b3",
        "resource",
        "metric/metricdata",
        "tag",
        "stats/internal",
        "stats",
        "internal/tagencoding",
        "metric/metricproducer",
        "stats/view",
        "plugin/ochttp",
        "plugin/ocgrpc",
    ],
    sum = "h1:75k/FF0Q2YM8QYo07VPddOLBslDt1MZOdEslOHvmzAs=",
    version = "v0.22.2",
    deps = [
        "//third_party/go/github.com/golang:groupcache",
        "//third_party/go/google.golang.org:grpc",
    ],
)
