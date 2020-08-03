subinclude("///pleasings2//go:compat")

go_library(
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
    srcs = glob(["*_test.go"]),
    deps = [":k8sclient"],
)

go_test(
    name = "integration_test",
    srcs = glob(["*_test.go"]),
    flags = "-test.run ^TestIntegration$",
    labels = ["integration"],
    deps = [":k8sclient"],
)
