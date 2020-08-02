subinclude("///pleasings2//go:compat")

go_library(
    name = "securityscan",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/anchore",
        "//internal/common",
        "//internal/integratedservices",
        "//internal/integratedservices/integratedserviceadapter",
        "//internal/integratedservices/services",
        "//internal/secret/secrettype",
        "//internal/security",
        "//pkg/backoff",
        "//pkg/k8sclient",
        "//pkg/security",
        "//src/auth",
        "//src/secret",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":securityscan",
        "//internal/anchore",
        "//internal/integratedservices",
        "//internal/integratedservices/services",
        "//src/secret",
    ],
)
