subinclude("///pleasings2//go:compat")

go_library(
    name = "restricted",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//src/secret"],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":restricted",
        "//internal/secret/secrettype",
        "//src/secret",
    ],
)
