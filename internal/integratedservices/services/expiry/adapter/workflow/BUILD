subinclude("///pleasings2//go:compat")

go_library(
    name = "workflow",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster",
        "//internal/integratedservices/services/expiry",
    ],
)
