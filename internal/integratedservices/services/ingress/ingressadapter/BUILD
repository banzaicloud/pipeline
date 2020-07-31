subinclude("///pleasings2//go:compat")

filegroup(
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
    srcs = [":ingressadapter"],
    deps = [
        "//internal/integratedservices/services/ingress",
        "//src/auth",
    ],
)
