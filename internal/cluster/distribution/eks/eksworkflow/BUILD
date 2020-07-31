subinclude("///pleasings2//go:compat")

filegroup(
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
        "//internal/providers/amazon",
        "//pkg/providers/amazon/cloudformation",
        "//pkg/sdk/brn",
        "//pkg/sdk/cadence/lib/pipeline/processlog",
    ],
)

go_test(
    name = "test",
    srcs = [":eksworkflow"],
    deps = [],
)
