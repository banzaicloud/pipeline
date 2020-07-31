subinclude("///pleasings2//go:compat")

filegroup(
    name = "helm",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//pkg/helm",
    ],
)

go_test(
    name = "test",
    srcs = [":helm"],
    deps = [
        "//internal/cmd",
        "//internal/common",
        "//internal/helm/helmadapter",
        "//internal/helm/testing",
        "//internal/testing",
        "//pkg/k8sclient",
    ],
)

go_test(
    name = "integration_test",
    srcs = [":helm"],
    flags = "-test.run ^TestIntegration$",
    labels = ["integration"],
    deps = [
        "//internal/cmd",
        "//internal/common",
        "//internal/helm/helmadapter",
        "//internal/helm/testing",
        "//internal/testing",
        "//pkg/k8sclient",
    ],
)
