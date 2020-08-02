subinclude("///pleasings2//go:compat")

go_library(
    name = "azure",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/providers",
        "//internal/secret/secrettype",
        "//pkg/objectstore",
        "//pkg/providers/azure",
        "//pkg/providers/azure/objectstore",
        "//src/secret",
    ],
)
