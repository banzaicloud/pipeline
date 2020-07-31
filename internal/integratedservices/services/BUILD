subinclude("///pleasings2//go:compat")

filegroup(
    name = "services",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//internal/helm",
        "//internal/integratedservices",
        "//pkg/helm",
    ],
)

go_test(
    name = "test",
    srcs = [":services"],
    deps = [
        "//internal/integratedservices",
        "//internal/integratedservices/services/expiry",
    ],
)
