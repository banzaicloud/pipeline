subinclude("///pleasings2//go:compat")

filegroup(
    name = "federation",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/clustergroup/api",
        "//internal/helm",
        "//internal/integratedservices/services/dns",
        "//pkg/k8sclient",
        "//src/cluster",
        "//src/dns",
    ],
)

go_test(
    name = "test",
    srcs = [":federation"],
    deps = [
        "//internal/cmd",
        "//internal/helm",
        "//internal/helm/testing",
        "//internal/testing",
        "//pkg/k8sclient",
    ],
)

go_test(
    name = "integration_test",
    srcs = [":federation"],
    flags = "-test.run ^TestIntegration$",
    labels = ["integration"],
    deps = [
        "//internal/cmd",
        "//internal/helm",
        "//internal/helm/testing",
        "//internal/testing",
        "//pkg/k8sclient",
    ],
)
