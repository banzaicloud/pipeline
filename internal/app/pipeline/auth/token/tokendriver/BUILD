subinclude("///pleasings2//go:compat")

filegroup(
    name = "tokendriver",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/app/pipeline/auth/token",
        "//internal/platform/appkit/transport/http",
    ],
)

go_test(
    name = "test",
    srcs = [":tokendriver"],
    deps = ["//internal/app/pipeline/auth/token"],
)
