subinclude("///pleasings2//go:compat")

filegroup(
    name = "expiry",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//internal/integratedservices",
    ],
)

go_test(
    name = "test",
    srcs = [":expiry"],
    deps = [],
)
