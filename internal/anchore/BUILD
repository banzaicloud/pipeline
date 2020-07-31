subinclude("///pleasings2//go:compat")

filegroup(
    name = "anchore",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [],
)

go_test(
    name = "test",
    srcs = [":anchore"],
    deps = [],
)
