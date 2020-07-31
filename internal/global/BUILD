subinclude("///pleasings2//go:compat")

filegroup(
    name = "global",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/platform/log"],
)
