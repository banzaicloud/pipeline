subinclude("///pleasings2//go:compat")

filegroup(
    name = "pkeaws",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/common"],
)

go_test(
    name = "test",
    srcs = [":pkeaws"],
    deps = [],
)
