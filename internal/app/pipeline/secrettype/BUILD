subinclude("///pleasings2//go:compat")

filegroup(
    name = "secrettype",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/secret"],
)

go_test(
    name = "test",
    srcs = [":secrettype"],
    deps = [],
)
