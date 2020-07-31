subinclude("///pleasings2//go:compat")

filegroup(
    name = "prometheus",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/cluster/metrics"],
)
