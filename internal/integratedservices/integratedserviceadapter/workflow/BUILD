subinclude("///pleasings2//go:compat")

filegroup(
    name = "workflow",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/integratedservices"],
)
