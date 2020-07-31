subinclude("///pleasings2//go:compat")

filegroup(
    name = "pkesecret",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//internal/secret/secrettype",
    ],
)
