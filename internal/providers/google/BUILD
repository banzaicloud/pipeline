subinclude("///pleasings2//go:compat")

filegroup(
    name = "google",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster/clusteradapter/clustermodel",
        "//internal/network",
        "//internal/objectstore",
        "//internal/secret/secrettype",
        "//pkg/gormhelper",
        "//pkg/objectstore",
        "//pkg/providers",
        "//pkg/providers/google",
        "//pkg/providers/google/objectstore",
        "//src/auth",
        "//src/secret",
    ],
)

go_test(
    name = "test",
    srcs = [":google"],
    deps = ["//internal/cluster/clusteradapter/clustermodel"],
)
