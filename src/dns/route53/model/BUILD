subinclude("///pleasings2//go:compat")

filegroup(
    name = "model",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//src/auth"],
)
