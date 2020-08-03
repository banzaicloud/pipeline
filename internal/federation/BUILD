subinclude("///pleasings2//go:compat")

go_library(
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
    srcs = glob(["*_test.go"]),
    deps = [
        ":federation",
        "//internal/cmd",
        "//internal/helm",
        "//internal/helm/testing",
        "//internal/testing",
        "//pkg/k8sclient",
    ],
)

go_test(
    name = "integration_test",
    srcs = glob(["*_test.go"]),
    flags = "-test.run ^TestIntegration$",
    labels = ["integration"],
    deps = [
        ":federation",
        "//internal/cmd",
        "//internal/helm",
        "//internal/helm/testing",
        "//internal/testing",
        "//pkg/k8sclient",
    ],
)
