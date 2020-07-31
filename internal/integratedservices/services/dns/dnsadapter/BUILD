subinclude("///pleasings2//go:compat")

filegroup(
    name = "dnsadapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//src/auth",
        "//src/cluster",
    ],
)

go_test(
    name = "test",
    srcs = [":dnsadapter"],
    deps = [
        "//internal/integratedservices/services",
        "//src/auth",
    ],
)
