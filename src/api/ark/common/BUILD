subinclude("///pleasings2//go:compat")

go_library(
    name = "common",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/ark",
        "//internal/cluster/clusteradapter",
        "//internal/global",
        "//internal/platform/gin/utils",
        "//internal/providers",
        "//internal/secret/restricted",
        "//pkg/common",
        "//src/auth",
        "//src/cluster",
    ],
)
