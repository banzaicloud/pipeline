subinclude("///pleasings2//go:compat")

go_library(
    name = "cloudformation",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/global"],
)
