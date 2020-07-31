subinclude("///pleasings2//go:compat")

filegroup(
    name = "ark",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/ark/api",
        "//internal/ark/client",
        "//internal/ark/providers/amazon",
        "//internal/ark/providers/azure",
        "//internal/ark/providers/google",
        "//internal/global",
        "//internal/providers",
        "//pkg/errors",
        "//pkg/providers",
        "//src/auth",
        "//src/model",
        "//src/secret",
    ],
)
