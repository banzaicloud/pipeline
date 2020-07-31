subinclude("///pleasings2//go:compat")

filegroup(
    name = "clustergroup",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/clustergroup",
        "//internal/clustergroup/api",
        "//internal/clustergroup/deployment",
        "//internal/platform/gin/utils",
        "//src/api/clustergroup/common",
        "//src/api/clustergroup/deployment",
        "//src/api/clustergroup/feature",
        "//src/auth",
    ],
)
