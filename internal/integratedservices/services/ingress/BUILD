subinclude("///pleasings2//go:compat")

filegroup(
    name = "ingress",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/integratedservices",
        "//internal/integratedservices/services",
        "//internal/providers/amazon",
        "//pkg/any",
        "//pkg/cluster",
        "//pkg/errors",
        "//pkg/jsonstructure",
        "//pkg/values",
    ],
)

go_test(
    name = "test",
    srcs = [":ingress"],
    deps = [
        "//internal/integratedservices",
        "//internal/integratedservices/services",
        "//pkg/jsonstructure",
    ],
)
