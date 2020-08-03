subinclude("///pleasings2//go:compat")

go_library(
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
    srcs = glob(["*_test.go"]),
    deps = [
        ":services",
        "//internal/integratedservices",
        "//internal/integratedservices/services/expiry",
    ],
)
