subinclude("///pleasings2//go:compat")

go_library(
    name = "gke",
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
