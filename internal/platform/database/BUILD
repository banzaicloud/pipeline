subinclude("///pleasings2//go:compat")

filegroup(
    name = "database",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [],
)

go_test(
    name = "test",
    srcs = [":database"],
    deps = [],
)
