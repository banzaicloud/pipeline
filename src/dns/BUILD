subinclude("///pleasings2//go:compat")

filegroup(
    name = "dns",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/global",
        "//internal/secret/secrettype",
        "//src/dns/route53",
    ],
)
