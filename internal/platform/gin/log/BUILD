subinclude("///pleasings2//go:compat")

filegroup(
    name = "log",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/platform/gin/correlationid"],
)
