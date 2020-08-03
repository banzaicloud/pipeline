subinclude("///pleasings2//go:compat")

go_library(
    name = "utils",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/platform/context",
        "//internal/platform/gin/correlationid",
        "//pkg/common",
    ],
)
