subinclude("///pleasings2//go:compat")

go_library(
    name = "auth",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster/clustersecret",
        "//internal/global",
        "//internal/secret/secrettype",
        "//src/secret",
    ],
)
