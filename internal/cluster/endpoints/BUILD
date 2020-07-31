subinclude("///pleasings2//go:compat")

filegroup(
    name = "endpoints",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//pkg/helm",
        "//pkg/k8sclient",
    ],
)

go_test(
    name = "test",
    srcs = [":endpoints"],
    deps = [
        "//internal/common",
        "//pkg/helm",
    ],
)
