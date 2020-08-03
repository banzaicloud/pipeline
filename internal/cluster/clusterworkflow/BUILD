subinclude("///pleasings2//go:compat")

go_library(
    name = "clusterworkflow",
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
        "//internal/global",
        "//pkg/cadence",
        "//pkg/kubernetes/custom/npls",
        "//pkg/providers",
        "//src/model",
    ],
)
