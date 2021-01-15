subinclude("///pleasings2//go:compat")

go_library(
    name = "eksworkflow",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster",
        "//internal/cluster/clusterworkflow",
        "//internal/cluster/distribution/eks",
        "//internal/cluster/distribution/eks/eksprovider/workflow",
        "//internal/providers/amazon",
        "//pkg/providers/amazon/cloudformation",
        "//pkg/sdk/brn",
        "//pkg/sdk/cadence/lib/pipeline/processlog",
    ],
)
