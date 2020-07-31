subinclude("///pleasings2//go:compat")

filegroup(
    name = "k8sclient",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/global"],
)

go_test(
    name = "test",
    srcs = [":k8sclient"],
    deps = [],
)

go_test(
    name = "integration_test",
    srcs = [":k8sclient"],
    flags = "-test.run ^TestIntegration$",
    labels = ["integration"],
    deps = [],
)
