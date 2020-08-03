subinclude("///pleasings2//go:compat")

go_library(
    name = "workflow",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster",
        "//internal/cluster/distribution",
        "//internal/cluster/distribution/eks",
        "//internal/cluster/distribution/eks/eksmodel",
        "//internal/global",
        "//internal/providers/amazon",
        "//internal/secret/secrettype",
        "//internal/secret/ssh/sshadapter",
        "//pkg/brn",
        "//pkg/cadence",
        "//pkg/cluster",
        "//pkg/k8sclient",
        "//pkg/providers/amazon",
        "//pkg/providers/amazon/autoscaling",
        "//pkg/providers/amazon/cloudformation",
        "//pkg/providers/amazon/ec2",
        "//src/secret",
        "//src/utils",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [":workflow"],
)
