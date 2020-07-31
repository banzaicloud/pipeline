subinclude("///pleasings2//go:compat")

filegroup(
    name = "nodepools",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//pkg/providers/amazon/autoscaling"],
)
