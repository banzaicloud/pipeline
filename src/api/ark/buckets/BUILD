subinclude("///pleasings2//go:compat")

go_library(
    name = "buckets",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/ark",
        "//internal/ark/api",
        "//internal/ark/sync",
        "//internal/global",
        "//internal/platform/gin/correlationid",
        "//internal/platform/gin/utils",
        "//pkg/providers",
        "//src/api/ark/common",
        "//src/auth",
    ],
)
