subinclude("///pleasings2//go:compat")

go_library(
    name = "securityscanadapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/integratedservices/services",
        "//internal/secret/secrettype",
        "//src/cluster",
        "//src/secret",
    ],
)
