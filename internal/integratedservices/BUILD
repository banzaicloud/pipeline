subinclude("///pleasings2//go:compat")

filegroup(
    name = "integratedservices",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/common"],
)

go_test(
    name = "test",
    srcs = [":integratedservices"],
    deps = [],
)
