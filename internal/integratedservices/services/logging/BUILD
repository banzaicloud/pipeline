subinclude("///pleasings2//go:compat")

go_library(
    name = "logging",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster/endpoints",
        "//internal/common",
        "//internal/helm",
        "//internal/integratedservices",
        "//internal/integratedservices/integratedserviceadapter",
        "//internal/integratedservices/services",
        "//internal/providers",
        "//internal/secret/secrettype",
        "//pkg/any",
        "//pkg/cluster",
        "//pkg/helm",
        "//pkg/jsonstructure",
        "//pkg/providers/azure",
        "//pkg/providers/azure/objectstore",
        "//src/auth",
        "//src/cluster",
        "//src/dns",
        "//src/secret",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":logging",
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
