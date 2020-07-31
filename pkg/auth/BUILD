subinclude("///pleasings2//go:compat")

filegroup(
    name = "auth",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [],
)

go_test(
    name = "test",
    srcs = [":auth"],
    deps = [],
)
