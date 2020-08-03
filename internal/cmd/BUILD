subinclude("///pleasings2//go:compat")

go_library(
    name = "cmd",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster/clusterconfig",
        "//internal/common",
        "//internal/federation",
        "//internal/helm",
        "//internal/helm/helmadapter",
        "//internal/integratedservices/services/dns",
        "//internal/integratedservices/services/ingress",
        "//internal/integratedservices/services/logging",
        "//internal/integratedservices/services/monitoring",
        "//internal/integratedservices/services/securityscan",
        "//internal/integratedservices/services/vault",
        "//internal/istio/istiofeature",
        "//internal/platform/cadence",
        "//internal/platform/database",
        "//internal/platform/log",
        "//pkg/cluster",
        "//pkg/values",
        "//src/cluster",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":cmd",
        "//internal/integratedservices/services/dns",
        "//internal/integratedservices/services/ingress",
        "//pkg/hook",
        "//pkg/values",
    ],
)
