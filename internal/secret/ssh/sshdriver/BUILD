subinclude("///pleasings2//go:compat")

filegroup(
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
