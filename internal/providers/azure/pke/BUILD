subinclude("///pleasings2//go:compat")

filegroup(
    name = "pke",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster/clusterbase",
        "//internal/pke",
        "//pkg/cluster",
    ],
)
