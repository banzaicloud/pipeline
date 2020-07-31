subinclude("///pleasings2//go:compat")

filegroup(
    name = "notification",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/common"],
)

go_test(
    name = "test",
    srcs = [":notification"],
    deps = [],
)
