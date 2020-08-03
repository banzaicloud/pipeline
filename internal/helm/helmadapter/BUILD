subinclude("///pleasings2//go:compat")

go_library(
    name = "helmadapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//internal/helm",
        "//internal/security",
        "//pkg/helm",
        "//pkg/k8sclient",
        "//src/auth",
        "//src/helm",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":helmadapter",
        "//internal/common",
        "//internal/helm",
    ],
)
