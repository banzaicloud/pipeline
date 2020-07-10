package(default_visibility = ["PUBLIC"])

go_get(
    name = "metrics",
    get = "go.uber.org/net/metrics",
    install = [
        "push",
        "",
        "tallypush",
        "bucket",
    ],
    sum = "h1:iRLPuVecNYf/wIV+mQaA4IgN8ghifu3q1B4IT6HfwyY=",
    version = "v1.3.0",
    deps = [
        "//third_party/go/github.com/prometheus:client_golang",
        "//third_party/go/github.com/prometheus:client_model",
        "//third_party/go/github.com/uber-go:tally",
        "//third_party/go/go.uber.org:atomic",
    ],
)
