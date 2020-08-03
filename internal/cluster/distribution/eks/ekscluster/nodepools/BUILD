subinclude("///pleasings2//go:compat")

go_library(
    name = "nodepools",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//pkg/providers/amazon/autoscaling"],
)
