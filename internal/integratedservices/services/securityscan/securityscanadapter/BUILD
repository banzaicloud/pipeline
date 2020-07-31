subinclude("///pleasings2//go:compat")

filegroup(
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
