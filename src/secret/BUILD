subinclude("///pleasings2//go:compat")

go_library(
    name = "secret",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/global",
        "//internal/secret",
        "//internal/secret/secrettype",
        "//pkg/secret",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [":secret"],
)
