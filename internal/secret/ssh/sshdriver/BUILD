subinclude("///pleasings2//go:compat")

go_library(
    name = "sshdriver",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/secret/secrettype",
        "//internal/secret/ssh",
        "//src/secret",
    ],
)
