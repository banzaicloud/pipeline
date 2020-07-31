subinclude("///pleasings2//go:compat")

filegroup(
    name = "providers",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//pkg/errors",
        "//pkg/providers/alibaba",
        "//pkg/providers/amazon",
        "//pkg/providers/azure",
        "//pkg/providers/google",
        "//pkg/providers/oracle",
        "//src/secret",
    ],
)
