subinclude("///pleasings2//go:compat")

filegroup(
    name = "backups",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/ark",
        "//internal/ark/api",
        "//internal/ark/clustermanager",
        "//internal/ark/sync",
        "//internal/global",
        "//internal/platform/gin/correlationid",
        "//internal/platform/gin/utils",
        "//src/api/ark/common",
        "//src/auth",
        "//src/cluster",
    ],
)
