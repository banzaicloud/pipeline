subinclude("///pleasings2//go:compat")

filegroup(
    name = "api",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//pkg/cluster",
        "//src/secret",
    ],
)
