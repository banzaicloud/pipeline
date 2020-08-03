subinclude("///pleasings2//go:compat")

go_library(
    name = "clustersetup",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster/clusterconfig",
        "//pkg/k8sutil",
        "//pkg/kubernetes",
        "//pkg/kubernetes/custom/npls",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":clustersetup",
        "//pkg/k8sclient",
        "//pkg/kubernetes",
    ],
)

go_test(
    name = "integration_test",
    srcs = glob(["*_test.go"]),
    flags = "-test.run ^TestIntegration$",
    labels = ["integration"],
    deps = [
        ":clustersetup",
        "//pkg/k8sclient",
        "//pkg/kubernetes",
    ],
)
