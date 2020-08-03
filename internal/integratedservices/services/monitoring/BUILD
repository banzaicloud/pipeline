subinclude("///pleasings2//go:compat")

go_library(
    name = "monitoring",
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
        "//internal/secret/secrettype",
        "//pkg/any",
        "//pkg/helm",
        "//pkg/jsonstructure",
        "//pkg/k8sclient",
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
        ":monitoring",
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
