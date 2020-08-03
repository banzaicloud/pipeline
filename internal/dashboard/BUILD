subinclude("///pleasings2//go:compat")

go_library(
    name = "dashboard",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster/auth",
        "//internal/cluster/oidc",
        "//internal/cluster/resourcesummary",
        "//internal/clustergroup",
        "//internal/platform/gin/utils",
        "//pkg/common",
        "//pkg/k8sclient",
        "//pkg/k8sutil",
        "//src/auth",
        "//src/cluster",
    ],
)
