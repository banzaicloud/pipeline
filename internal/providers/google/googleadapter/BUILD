subinclude("///pleasings2//go:compat")

filegroup(
    name = "googleadapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//internal/providers/google",
        "//internal/secret/secrettype",
    ],
)
