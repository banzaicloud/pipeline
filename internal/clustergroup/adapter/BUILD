subinclude("///pleasings2//go:compat")

filegroup(
    name = "adapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/clustergroup/api",
        "//src/cluster",
    ],
)
