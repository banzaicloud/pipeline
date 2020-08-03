subinclude("///pleasings2//go:compat")

go_library(
    name = "clusteradapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster",
        "//internal/cluster/clusteradapter/clustermodel",
        "//internal/cluster/clusterworkflow",
        "//internal/cluster/distribution/eks",
        "//internal/cluster/distribution/eks/eksmodel",
        "//pkg/brn",
        "//pkg/cloudinfo",
        "//pkg/providers",
        "//src/model",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":clusteradapter",
        "//internal/cluster",
        "//internal/cluster/clusterworkflow",
        "//internal/cluster/distribution/eks",
    ],
)
