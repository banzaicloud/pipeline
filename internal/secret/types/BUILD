subinclude("///pleasings2//go:compat")

go_library(
    name = "types",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/secret",
        "//pkg/k8sclient",
        "//pkg/providers/azure",
        "//pkg/providers/oracle/oci",
        "//pkg/secret",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":types",
        "//internal/secret",
    ],
)
