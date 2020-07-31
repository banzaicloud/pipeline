subinclude("///pleasings2//go:compat")

filegroup(
    name = "route53",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/global",
        "//internal/secret/secrettype",
        "//pkg/cluster",
        "//pkg/providers/amazon",
        "//src/auth",
        "//src/dns/route53/model",
        "//src/secret",
    ],
)

go_test(
    name = "test",
    srcs = [":route53"],
    deps = [
        "//internal/common",
        "//internal/global",
        "//internal/secret/pkesecret",
        "//internal/secret/restricted",
        "//internal/secret/secretadapter",
        "//internal/secret/secrettype",
        "//internal/secret/types",
        "//pkg/cluster",
        "//src/auth",
        "//src/secret",
    ],
)

go_test(
    name = "integration_test",
    srcs = [":route53"],
    flags = "-test.run ^TestIntegration$",
    labels = ["integration"],
    deps = [
        "//internal/common",
        "//internal/global",
        "//internal/secret/pkesecret",
        "//internal/secret/restricted",
        "//internal/secret/secretadapter",
        "//internal/secret/secrettype",
        "//internal/secret/types",
        "//pkg/cluster",
        "//src/auth",
        "//src/secret",
    ],
)
