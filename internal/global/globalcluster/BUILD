subinclude("///pleasings2//go:compat")

filegroup(
    name = "globalcluster",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/cluster"],
)
