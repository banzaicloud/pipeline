subinclude("///pleasings2//go:compat")

filegroup(
    name = "sync",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/ark",
        "//internal/ark/api",
        "//pkg/cluster",
        "//src/auth",
    ],
)
