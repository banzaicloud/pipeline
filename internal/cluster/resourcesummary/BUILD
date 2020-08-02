subinclude("///pleasings2//go:compat")

go_library(
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
    srcs = glob(["*_test.go"]),
    deps = [":resourcesummary"],
)
