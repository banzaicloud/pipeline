subinclude("///pleasings2//go:compat")

filegroup(
    name = "brn",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [],
)

go_test(
    name = "test",
    srcs = [":brn"],
    deps = [],
)
