subinclude("///pleasings2//go:compat")

filegroup(
    name = "resourcesummary",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//pkg/k8sutil"],
)

go_test(
    name = "test",
    srcs = [":resourcesummary"],
    deps = [],
)
