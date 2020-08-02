subinclude("///pleasings2//go:compat")

go_library(
    name = "azure",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/secret/secrettype",
        "//pkg/cluster",
    ],
)
