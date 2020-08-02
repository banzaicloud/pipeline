subinclude("///pleasings2//go:compat")

go_library(
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
    srcs = glob(["*_test.go"]),
    deps = [
        ":endpoints",
        "//internal/common",
        "//pkg/helm",
    ],
)
