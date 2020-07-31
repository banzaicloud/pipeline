subinclude("///pleasings2//go:compat")

filegroup(
    name = "namespace",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//pkg/brn",
        "//pkg/common",
        "//src/api/common",
    ],
)
