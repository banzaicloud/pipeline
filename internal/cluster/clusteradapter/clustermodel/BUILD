subinclude("///pleasings2//go:compat")

filegroup(
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
