subinclude("///pleasings2//go:compat")

filegroup(
    name = "oracle",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/network",
        "//internal/objectstore",
        "//internal/secret/secrettype",
        "//pkg/objectstore",
        "//pkg/providers",
        "//pkg/providers/oracle",
        "//pkg/providers/oracle/model",
        "//pkg/providers/oracle/objectstore",
        "//pkg/providers/oracle/oci",
        "//pkg/providers/oracle/secret",
        "//src/auth",
        "//src/secret",
    ],
)
