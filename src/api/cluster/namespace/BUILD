subinclude("///pleasings2//go:compat")

go_library(
    name = "namespace",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//pkg/sdk/brn",
        "//pkg/common",
        "//src/api/common",
    ],
)
