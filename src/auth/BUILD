subinclude("///pleasings2//go:compat")

filegroup(
    name = "auth",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//internal/global",
        "//pkg/auth",
        "//pkg/common",
    ],
)

go_test(
    name = "test",
    srcs = [":auth"],
    deps = ["//internal/common"],
)

go_test(
    name = "integration_test",
    srcs = [":auth"],
    flags = "-test.run ^TestIntegration$",
    labels = ["integration"],
    deps = ["//internal/common"],
)
