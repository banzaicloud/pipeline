subinclude("///pleasings2//go:compat")

go_library(
    name = "model",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster/clusteradapter/clustermodel",
        "//internal/database/sql/json",
        "//internal/global",
        "//internal/providers/azure/azureadapter",
        "//internal/providers/kubernetes/kubernetesadapter",
        "//pkg/cluster",
    ],
)
