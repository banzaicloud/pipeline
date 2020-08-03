subinclude("///pleasings2//go:compat")

go_library(
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
