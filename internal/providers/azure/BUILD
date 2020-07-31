subinclude("///pleasings2//go:compat")

filegroup(
    name = "azure",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/global",
        "//internal/network",
        "//internal/objectstore",
        "//internal/secret/secrettype",
        "//pkg/objectstore",
        "//pkg/providers",
        "//pkg/providers/azure",
        "//pkg/providers/azure/objectstore",
        "//src/auth",
        "//src/secret",
    ],
)
