subinclude("///pleasings2//go:compat")

filegroup(
    name = "deployment",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/clustergroup/api",
        "//internal/global",
        "//internal/helm",
        "//pkg/jsonstructure",
        "//src/helm",
    ],
)

go_test(
    name = "test",
    srcs = [":deployment"],
    deps = [
        "//internal/cmd",
        "//internal/common",
        "//internal/global",
        "//internal/helm",
        "//internal/helm/testing",
    ],
)

go_test(
    name = "integration_test",
    srcs = [":deployment"],
    flags = "-test.run ^TestIntegration$",
    labels = ["integration"],
    deps = [
        "//internal/cmd",
        "//internal/common",
        "//internal/global",
        "//internal/helm",
        "//internal/helm/testing",
    ],
)
