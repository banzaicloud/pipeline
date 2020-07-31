subinclude("///pleasings2//go:compat")

filegroup(
    name = "auditlogdriver",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//internal/platform/gin/auditlog",
    ],
)

go_test(
    name = "test",
    srcs = [":auditlogdriver"],
    deps = [
        "//internal/common",
        "//internal/platform/gin/auditlog",
    ],
)
