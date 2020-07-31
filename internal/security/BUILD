subinclude("///pleasings2//go:compat")

filegroup(
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
