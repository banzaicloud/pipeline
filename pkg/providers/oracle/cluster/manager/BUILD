subinclude("///pleasings2//go:compat")

filegroup(
    name = "manager",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//pkg/common",
        "//pkg/providers/oracle/model",
        "//pkg/providers/oracle/oci",
    ],
)
