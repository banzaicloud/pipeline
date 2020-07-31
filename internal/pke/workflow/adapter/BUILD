subinclude("///pleasings2//go:compat")

filegroup(
    name = "adapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/pke/workflow",
        "//internal/secret/secrettype",
        "//pkg/brn",
    ],
)
