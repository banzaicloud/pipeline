subinclude("///pleasings2//go:compat")

filegroup(
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
