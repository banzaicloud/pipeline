subinclude("///pleasings2//go:compat")

go_library(
    name = "adapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//internal/integratedservices/services/expiry",
        "//internal/integratedservices/services/expiry/adapter/workflow",
    ],
)
