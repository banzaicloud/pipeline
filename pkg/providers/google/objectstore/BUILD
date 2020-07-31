subinclude("///pleasings2//go:compat")

filegroup(
    name = "objectstore",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//pkg/errors"],
)

go_test(
    name = "test",
    srcs = [":objectstore"],
    deps = [],
)

go_test(
    name = "integration_test",
    srcs = [":objectstore"],
    flags = "-test.run ^TestIntegration$",
    labels = ["integration"],
    deps = [],
)
