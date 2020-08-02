subinclude("///pleasings2//go:compat")

go_library(
    name = "secretadapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/secret"],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":secretadapter",
        "//internal/secret",
    ],
)

go_test(
    name = "integration_test",
    srcs = glob(["*_test.go"]),
    flags = "-test.run ^TestIntegration$",
    labels = ["integration"],
    deps = [
        ":secretadapter",
        "//internal/secret",
    ],
)
