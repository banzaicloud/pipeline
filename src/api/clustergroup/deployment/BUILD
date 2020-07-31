subinclude("///pleasings2//go:compat")

filegroup(
    name = "deployment",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/clustergroup",
        "//internal/clustergroup/deployment",
        "//internal/platform/gin/utils",
        "//pkg/common",
        "//src/api/clustergroup/common",
        "//src/auth",
    ],
)
