subinclude("///pleasings2//go:compat")

filegroup(
    name = "secretadapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/secret"],
)

go_test(
    name = "test",
    srcs = [":secretadapter"],
    deps = ["//internal/secret"],
)

go_test(
    name = "integration_test",
    srcs = [":secretadapter"],
    flags = "-test.run ^TestIntegration$",
    labels = ["integration"],
    deps = ["//internal/secret"],
)
