subinclude("///pleasings2//go:compat")

filegroup(
    name = "k8sutil",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//pkg/helm/kube"],
)

go_test(
    name = "test",
    srcs = [":k8sutil"],
    deps = [],
)
