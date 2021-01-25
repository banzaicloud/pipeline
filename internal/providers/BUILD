subinclude("///pleasings2//go:compat")

go_library(
    name = "providers",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common/commonadapter",
        "//internal/global",
        "//internal/network",
        "//internal/objectstore",
        "//internal/providers/amazon",
        "//internal/providers/azure",
        "//internal/providers/azure/pke/adapter",
        "//internal/providers/google",
        "//internal/providers/pke",
        "//internal/providers/vsphere/pke/adapter",
        "//pkg/errors",
        "//pkg/providers",
        "//src/auth",
        "//src/secret",
    ],
)
