subinclude("///pleasings2//go:compat")

go_library(
    name = "pkeworkflow",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster/auth",
        "//internal/cluster/clustersecret",
        "//internal/cluster/distribution",
        "//internal/cluster/distribution/eks/eksprovider/workflow",
        "//internal/cluster/distribution/pke/pkeaws",
        "//internal/providers/amazon",
        "//internal/providers/pke",
        "//internal/secret/secrettype",
        "//pkg/cluster/pke",
        "//pkg/k8sclient",
        "//pkg/providers/amazon",
        "//pkg/providers/amazon/cloudformation",
        "//pkg/providers/amazon/ec2",
        "//src/secret",
    ],
)
