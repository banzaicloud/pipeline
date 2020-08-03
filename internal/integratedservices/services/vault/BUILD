subinclude("///pleasings2//go:compat")

go_library(
    name = "vault",
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
        "//pkg/backoff",
        "//src/auth",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":vault",
        "//internal/common/commonadapter",
        "//internal/helm",
        "//internal/integratedservices",
        "//internal/integratedservices/integratedserviceadapter",
        "//internal/integratedservices/services",
        "//internal/secret/secrettype",
        "//pkg/cluster",
        "//pkg/helm",
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
        ":vault",
        "//internal/common/commonadapter",
        "//internal/helm",
        "//internal/integratedservices",
        "//internal/integratedservices/integratedserviceadapter",
        "//internal/integratedservices/services",
        "//internal/secret/secrettype",
        "//pkg/cluster",
        "//pkg/helm",
        "//src/auth",
        "//src/secret",
    ],
)
