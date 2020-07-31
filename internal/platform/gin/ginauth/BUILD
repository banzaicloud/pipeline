subinclude("///pleasings2//go:compat")

filegroup(
    name = "ginauth",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//src/auth"],
)

go_test(
    name = "test",
    srcs = [":ginauth"],
    deps = ["//src/auth"],
)
