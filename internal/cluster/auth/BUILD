subinclude("///pleasings2//go:compat")

filegroup(
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
