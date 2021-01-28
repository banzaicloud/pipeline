subinclude("///pleasings2//go:compat")

go_library(
    name = "providers",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//pkg/errors",
        "//pkg/providers/amazon",
        "//pkg/providers/azure",
        "//pkg/providers/google",
        "//src/secret",
    ],
)
