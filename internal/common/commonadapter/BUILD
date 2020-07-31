subinclude("///pleasings2//go:compat")

filegroup(
    name = "commonadapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//pkg/brn",
        "//src/secret",
    ],
)

go_test(
    name = "test",
    srcs = [":commonadapter"],
    deps = [
        "//internal/common",
        "//internal/secret/secrettype",
        "//src/secret",
    ],
)
