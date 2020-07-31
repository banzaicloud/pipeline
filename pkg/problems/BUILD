subinclude("///pleasings2//go:compat")

filegroup(
    name = "problems",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [],
)

go_test(
    name = "test",
    srcs = [":problems"],
    deps = [],
)
