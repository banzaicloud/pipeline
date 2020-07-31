subinclude("///pleasings2//go:compat")

filegroup(
    name = "adapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster/clusteradapter/clustermodel",
        "//internal/common",
        "//internal/database/sql/json",
        "//internal/pke",
        "//internal/providers/azure/pke",
        "//pkg/cluster",
    ],
)

go_test(
    name = "test",
    srcs = [":adapter"],
    deps = [],
)
