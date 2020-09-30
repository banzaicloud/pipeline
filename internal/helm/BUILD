subinclude("///pleasings2//go:compat")

go_library(
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
    srcs = glob(["*_test.go"], exclude = [
        "integration_test.go",
        "*_integration_test.go",
        "zz_generated.mock_external_test.go",
    ]),
    deps = [
        ":helm",
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
    srcs = glob([
        "integration_test.go",
        "*_integration_test.go",
        "zz_generated.mock_external_test.go",
    ]),
    flags = "-test.run ^TestIntegration$",
    labels = ["integration"],
    deps = [
        ":helm",
        "//internal/cmd",
        "//internal/common",
        "//internal/helm/helmadapter",
        "//internal/helm/testing",
        "//internal/testing",
        "//pkg/k8sclient",
    ],
)
