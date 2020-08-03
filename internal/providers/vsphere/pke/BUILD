subinclude("///pleasings2//go:compat")

go_library(
    name = "pke",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster/clusterbase",
        "//internal/global",
        "//internal/pke",
        "//pkg/cluster",
        "//pkg/cluster/pke",
    ],
)
