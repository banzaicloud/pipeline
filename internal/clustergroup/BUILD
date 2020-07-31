subinclude("///pleasings2//go:compat")

filegroup(
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
