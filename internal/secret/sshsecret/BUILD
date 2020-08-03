subinclude("///pleasings2//go:compat")

go_library(
    name = "sshsecret",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/secret/ssh",
        "//internal/secret/ssh/sshadapter",
        "//internal/secret/ssh/sshdriver",
        "//src/secret",
    ],
)
