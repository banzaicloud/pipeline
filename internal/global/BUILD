subinclude("///pleasings2//go:compat")

go_library(
    name = "global",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/platform/log"],
)
