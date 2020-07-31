subinclude("///pleasings2//go:compat")

filegroup(
    name = "clusterbase",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//pkg/cluster"],
)
