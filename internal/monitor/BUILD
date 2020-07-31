subinclude("///pleasings2//go:compat")

filegroup(
    name = "monitor",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//pkg/cluster",
        "//pkg/providers/amazon",
        "//pkg/providers/amazon/ec2",
        "//src/cluster",
    ],
)
