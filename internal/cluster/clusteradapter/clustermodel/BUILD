subinclude("///pleasings2//go:compat")

go_library(
    name = "clustermodel",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/database/sql/json",
        "//internal/global",
        "//pkg/gormhelper",
        "//src/secret",
    ],
)
