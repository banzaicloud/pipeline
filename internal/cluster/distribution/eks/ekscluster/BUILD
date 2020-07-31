subinclude("///pleasings2//go:compat")

filegroup(
    name = "ekscluster",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster/distribution/eks",
        "//internal/global/globaleks",
        "//pkg/common",
        "//pkg/errors",
    ],
)
