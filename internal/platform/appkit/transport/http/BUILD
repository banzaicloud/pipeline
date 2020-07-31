subinclude("///pleasings2//go:compat")

filegroup(
    name = "http",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//pkg/problems"],
)

go_test(
    name = "test",
    srcs = [":http"],
    deps = ["//pkg/problems"],
)
