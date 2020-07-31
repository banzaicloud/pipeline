subinclude("///pleasings2//go:compat")

filegroup(
    name = "token",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [],
)

go_test(
    name = "test",
    srcs = [":token"],
    deps = [],
)
