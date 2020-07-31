subinclude("///pleasings2//go:compat")

filegroup(
    name = "kubernetes",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//internal/secret/secrettype",
        "//pkg/k8sclient",
        "//pkg/kubernetes",
    ],
)
