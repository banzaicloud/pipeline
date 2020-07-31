subinclude("///pleasings2//go:compat")

filegroup(
    name = "restricted",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//src/secret"],
)

go_test(
    name = "test",
    srcs = [":restricted"],
    deps = [
        "//internal/secret/secrettype",
        "//src/secret",
    ],
)
