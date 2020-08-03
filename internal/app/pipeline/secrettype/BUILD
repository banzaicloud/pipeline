subinclude("///pleasings2//go:compat")

go_library(
    name = "secrettype",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/secret"],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [":secrettype"],
)
