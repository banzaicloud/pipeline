subinclude("///pleasings2//go:compat")

filegroup(
    name = "kubernetes",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//pkg/k8sclient"],
)
