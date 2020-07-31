subinclude("///pleasings2//go:compat")

filegroup(
    name = "kubesecret",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/secret/secrettype",
        "//src/secret",
    ],
)

go_test(
    name = "test",
    srcs = [":kubesecret"],
    deps = [],
)
