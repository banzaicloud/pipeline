subinclude("///pleasings2//go:compat")

filegroup(
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
    srcs = [":clustersetup"],
    deps = [
        "//pkg/k8sclient",
        "//pkg/kubernetes",
    ],
)

go_test(
    name = "integration_test",
    srcs = [":clustersetup"],
    flags = "-test.run ^TestIntegration$",
    labels = ["integration"],
    deps = [
        "//pkg/k8sclient",
        "//pkg/kubernetes",
    ],
)
