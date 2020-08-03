subinclude("///pleasings2//go:compat")

go_library(
    name = "pkeworkflowadapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/providers/pke/pkeworkflow",
        "//src/cluster",
        "//src/secret",
    ],
)
