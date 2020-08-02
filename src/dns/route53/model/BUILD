subinclude("///pleasings2//go:compat")

go_library(
    name = "model",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//src/auth"],
)
