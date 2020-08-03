subinclude("///pleasings2//go:compat")

go_library(
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
    srcs = glob(["*_test.go"]),
    deps = [
        ":route53",
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
    srcs = glob(["*_test.go"]),
    flags = "-test.run ^TestIntegration$",
    labels = ["integration"],
    deps = [
        ":route53",
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
