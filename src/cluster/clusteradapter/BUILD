subinclude("///pleasings2//go:compat")

filegroup(
    name = "clusteradapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster",
        "//src/cluster",
    ],
)
