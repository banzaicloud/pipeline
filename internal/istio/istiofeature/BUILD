subinclude("///pleasings2//go:compat")

go_library(
    name = "istiofeature",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/clustergroup/api",
        "//internal/helm",
        "//pkg/backoff",
        "//pkg/k8sclient",
        "//src/cluster",
    ],
)
