subinclude("///pleasings2//go:compat")

go_library(
    name = "prometheus",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/cluster/metrics"],
)
