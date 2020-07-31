subinclude("///pleasings2//go:compat")

filegroup(
    name = "eksadapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster",
        "//internal/cluster/distribution/eks",
        "//internal/cluster/distribution/eks/eksmodel",
        "//internal/cluster/distribution/eks/eksprovider/workflow",
        "//internal/cluster/distribution/eks/eksworkflow",
        "//pkg/kubernetes/custom/npls",
    ],
)

go_test(
    name = "test",
    srcs = [":eksadapter"],
    deps = [
        "//internal/cluster",
        "//internal/cluster/distribution/eks",
        "//internal/cluster/distribution/eks/eksmodel",
        "//internal/cluster/distribution/eks/eksprovider/workflow",
        "//pkg/brn",
    ],
)
