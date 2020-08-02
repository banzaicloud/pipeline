subinclude("///pleasings2//go:compat")

go_library(
    name = "objectstore",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//pkg/errors",
        "//pkg/providers/oracle/oci",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [":objectstore"],
)

go_test(
    name = "integration_test",
    srcs = glob(["*_test.go"]),
    flags = "-test.run ^TestIntegration$",
    labels = ["integration"],
    deps = [":objectstore"],
)
