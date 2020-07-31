subinclude("///pleasings2//go:compat")

filegroup(
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
