subinclude("///pleasings2//go:compat")

go_library(
    name = "common",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/platform/gin/correlationid",
        "//internal/platform/gin/utils",
        "//pkg/common",
        "//src/auth",
        "//src/cluster",
    ],
)
