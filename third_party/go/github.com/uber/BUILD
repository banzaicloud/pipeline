package(default_visibility = ["PUBLIC"])

go_get(
    name = "tchannel-go",
    get = "github.com/uber/tchannel-go",
    install = [
        "internal/argreader",
        "thrift/arg2",
        "relay",
        "tnet",
        "tos",
        "trand",
        "typed",
        "",
    ],
    sum = "h1:2Hxvd+LaQyb9Rza+KoobJU6lhabIo1+cImk/rdLfiuQ=",
    version = "v1.18.0",
    deps = [
        "//third_party/go/github.com/opentracing:opentracing-go",
        "//third_party/go/go.uber.org:atomic",
        "//third_party/go/go.uber.org:multierr",
        "//third_party/go/golang.org/x:net",
        "//third_party/go/golang.org/x:sys",
    ],
)
