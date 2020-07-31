subinclude("///pleasings2//go:compat")

filegroup(
    name = "secret",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/global",
        "//internal/secret",
        "//internal/secret/secrettype",
        "//pkg/secret",
    ],
)

go_test(
    name = "test",
    srcs = [":secret"],
    deps = [],
)
