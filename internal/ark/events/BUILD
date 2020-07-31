subinclude("///pleasings2//go:compat")

filegroup(
    name = "events",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/ark"],
)

go_test(
    name = "test",
    srcs = [":events"],
    deps = ["//src/cluster"],
)
