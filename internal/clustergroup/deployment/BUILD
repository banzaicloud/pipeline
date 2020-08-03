subinclude("///pleasings2//go:compat")

go_library(
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
    srcs = glob(["*_test.go"]),
    deps = [
        ":deployment",
        "//internal/cmd",
        "//internal/common",
        "//internal/global",
        "//internal/helm",
        "//internal/helm/testing",
    ],
)

go_test(
    name = "integration_test",
    srcs = glob(["*_test.go"]),
    flags = "-test.run ^TestIntegration$",
    labels = ["integration"],
    deps = [
        ":deployment",
        "//internal/cmd",
        "//internal/common",
        "//internal/global",
        "//internal/helm",
        "//internal/helm/testing",
    ],
)
