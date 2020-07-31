subinclude("///pleasings2//go:compat")

filegroup(
    name = "driver",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster",
        "//internal/cluster/distribution/eks",
        "//internal/cluster/distribution/eks/ekscluster",
        "//internal/cluster/distribution/eks/eksmodel",
        "//internal/cluster/distribution/eks/eksprovider/workflow",
        "//internal/cluster/metrics",
        "//internal/global",
        "//internal/secret/ssh",
        "//internal/secret/ssh/sshdriver",
        "//pkg/cluster",
        "//pkg/errors",
        "//pkg/providers/amazon",
        "//pkg/providers/amazon/ec2",
        "//src/auth",
        "//src/cluster",
        "//src/secret",
    ],
)

go_test(
    name = "test",
    srcs = [":driver"],
    deps = [
        "//internal/cluster/distribution/eks/ekscluster",
        "//internal/cluster/distribution/eks/eksprovider/workflow",
    ],
)
