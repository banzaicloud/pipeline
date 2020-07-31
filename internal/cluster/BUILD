subinclude("///pleasings2//go:compat")

filegroup(
    name = "cluster",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//pkg/brn",
        "//pkg/cloud",
    ],
)

go_test(
    name = "test",
    srcs = [":cluster"],
    deps = [
        "//pkg/brn",
        "//pkg/cloud",
    ],
)
