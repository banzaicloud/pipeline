subinclude("///pleasings2//go:compat")

go_library(
    name = "model",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/global",
        "//pkg/cluster",
        "//pkg/errors",
        "//pkg/providers/oracle/cluster",
    ],
)
