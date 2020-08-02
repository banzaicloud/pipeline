subinclude("///pleasings2//go:compat")

go_library(
    name = "clustergroup",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/clustergroup/api",
        "//internal/clustergroup/deployment",
        "//pkg/cluster",
    ],
)
