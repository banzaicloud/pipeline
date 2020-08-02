subinclude("///pleasings2//go:compat")

go_library(
    name = "globalcluster",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/cluster"],
)
