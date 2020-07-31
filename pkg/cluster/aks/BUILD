subinclude("///pleasings2//go:compat")

filegroup(
    name = "aks",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//pkg/common",
        "//pkg/errors",
    ],
)
