subinclude("///pleasings2//go:compat")

go_library(
    name = "expiry",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//internal/integratedservices",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [":expiry"],
)
