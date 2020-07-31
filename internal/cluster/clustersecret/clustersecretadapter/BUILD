subinclude("///pleasings2//go:compat")

filegroup(
    name = "clustersecretadapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster/clustersecret",
        "//src/cluster",
        "//src/secret",
    ],
)
