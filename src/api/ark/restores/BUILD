subinclude("///pleasings2//go:compat")

filegroup(
    name = "restores",
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
        "//src/api/ark/common",
    ],
)
