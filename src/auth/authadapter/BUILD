subinclude("///pleasings2//go:compat")

filegroup(
    name = "authadapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//src/auth"],
)

go_test(
    name = "test",
    srcs = [":authadapter"],
    deps = [
        "//internal/common",
        "//src/auth",
    ],
)
