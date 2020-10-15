subinclude("///pleasings2//go:compat")

go_library(
    name = "dns",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//internal/integratedservices",
        "//internal/integratedservices/integratedserviceadapter",
        "//internal/integratedservices/services",
        "//internal/integratedservices/services/dns/externaldns",
        "//internal/secret/secrettype",
        "//pkg/any",
        "//pkg/sdk/brn",
        "//pkg/validation",
        "//src/auth",
        "//src/cluster",
        "//src/dns/route53",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":dns",
        "//internal/common/commonadapter",
        "//internal/helm",
        "//internal/integratedservices",
        "//internal/integratedservices/integratedserviceadapter",
        "//internal/integratedservices/services",
        "//internal/secret/secrettype",
        "//pkg/sdk/brn",
        "//pkg/cluster",
        "//pkg/helm",
        "//src/auth",
        "//src/dns/route53",
        "//src/secret",
    ],
)
