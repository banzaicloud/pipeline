subinclude("///pleasings2//go:compat")

go_library(
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
    srcs = glob(["*_test.go"]),
    deps = [
        ":integratedservicesdriver",
        "//.gen/pipeline/pipeline",
        "//internal/integratedservices",
    ],
)
