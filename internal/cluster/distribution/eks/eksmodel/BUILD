subinclude("///pleasings2//go:compat")

go_library(
    name = "eksmodel",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster/clusteradapter/clustermodel",
        "//internal/global",
        "//pkg/gormhelper",
    ],
)
