subinclude("///pleasings2//go:compat")

go_library(
    name = "feature",
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
        "//pkg/common",
        "//src/api/clustergroup/common",
        "//src/auth",
    ],
)
