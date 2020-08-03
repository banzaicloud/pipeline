subinclude("///pleasings2//go:compat")

go_library(
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
    srcs = glob(["*_test.go"]),
    deps = [":k8sutil"],
)
