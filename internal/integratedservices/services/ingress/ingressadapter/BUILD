subinclude("///pleasings2//go:compat")

go_library(
    name = "ingressadapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster",
        "//internal/integratedservices/services/ingress",
        "//src/auth",
        "//src/dns",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":ingressadapter",
        "//internal/integratedservices/services/ingress",
        "//src/auth",
    ],
)
