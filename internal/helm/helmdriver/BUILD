subinclude("///pleasings2//go:compat")

filegroup(
    name = "helmdriver",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//.gen/pipeline/pipeline",
        "//internal/helm",
        "//internal/platform/appkit/transport/http",
        "//pkg/helm",
    ],
)

go_test(
    name = "test",
    srcs = [":helmdriver"],
    deps = ["//internal/helm"],
)
