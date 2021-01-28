subinclude("///pleasings2//go:compat")

go_library(
    name = "cluster",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster/distribution/eks/ekscluster",
        "//pkg/cluster/aks",
        "//pkg/cluster/gke",
        "//pkg/cluster/kubernetes",
        "//pkg/cluster/pke",
        "//pkg/common",
        "//pkg/errors",
        "//pkg/values",
    ],
)
