subinclude("///pleasings2//go:compat")

go_library(
    name = "adapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/pke/workflow",
        "//internal/secret/secrettype",
        "//pkg/sdk/brn",
    ],
)
