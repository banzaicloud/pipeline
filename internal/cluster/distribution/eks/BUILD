subinclude("///pleasings2//go:compat")

filegroup(
    name = "eks",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/cluster"],
)

go_test(
    name = "test",
    srcs = [":eks"],
    deps = [
        "//internal/cluster",
        "//pkg/brn",
    ],
)
