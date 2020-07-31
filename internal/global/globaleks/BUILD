subinclude("///pleasings2//go:compat")

filegroup(
    name = "globaleks",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/cluster/distribution/eks"],
)
