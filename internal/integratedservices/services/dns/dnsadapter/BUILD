subinclude("///pleasings2//go:compat")

go_library(
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
    srcs = glob(["*_test.go"]),
    deps = [
        ":dnsadapter",
        "//internal/integratedservices/services",
        "//src/auth",
    ],
)
