subinclude("///pleasings2//go:compat")

filegroup(
    name = "verify",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/secret/secrettype",
        "//pkg/k8sclient",
    ],
)
