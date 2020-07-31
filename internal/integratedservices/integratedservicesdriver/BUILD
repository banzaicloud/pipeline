subinclude("///pleasings2//go:compat")

filegroup(
    name = "integratedservicesdriver",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//.gen/pipeline/pipeline",
        "//internal/integratedservices",
        "//internal/platform/appkit/transport/http",
    ],
)

go_test(
    name = "test",
    srcs = [":integratedservicesdriver"],
    deps = [
        "//.gen/pipeline/pipeline",
        "//internal/integratedservices",
    ],
)
