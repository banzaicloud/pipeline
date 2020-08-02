subinclude("///pleasings2//go:compat")

go_library(
    name = "security",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//.gen/anchore",
        "//internal/anchore",
        "//internal/common",
        "//pkg/k8sclient",
        "//pkg/security",
        "//src/secret",
    ],
)
