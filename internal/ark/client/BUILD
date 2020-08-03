subinclude("///pleasings2//go:compat")

go_library(
    name = "client",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/ark/api",
        "//pkg/k8sclient",
    ],
)
