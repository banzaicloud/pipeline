subinclude("///pleasings2//go:compat")

filegroup(
    name = "auditlog",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//internal/platform/gin/correlationid",
    ],
)

go_test(
    name = "test",
    srcs = [":auditlog"],
    deps = ["//internal/platform/gin/correlationid"],
)
