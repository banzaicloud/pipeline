subinclude("///pleasings2//go:compat")

go_library(
    name = "pke",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster/clusteradapter/clustermodel",
        "//internal/database/sql/json",
        "//pkg/providers/amazon",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [":pke"],
)
