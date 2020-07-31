subinclude("///pleasings2//go:compat")

filegroup(
    name = "sshadapter",
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
