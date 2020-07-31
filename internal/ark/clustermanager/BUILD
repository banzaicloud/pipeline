subinclude("///pleasings2//go:compat")

filegroup(
    name = "clustermanager",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/ark/api",
        "//src/cluster",
    ],
)
