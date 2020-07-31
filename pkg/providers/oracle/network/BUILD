subinclude("///pleasings2//go:compat")

filegroup(
    name = "network",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//pkg/providers/oracle/oci"],
)
