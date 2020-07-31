subinclude("///pleasings2//go:compat")

filegroup(
    name = "backupservice",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/ark/api",
        "//internal/global",
        "//internal/helm",
        "//internal/platform/gin/correlationid",
        "//pkg/common",
        "//pkg/providers",
        "//src/api/ark/common",
        "//src/auth",
    ],
)
