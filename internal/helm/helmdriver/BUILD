subinclude("///pleasings2//go:compat")

go_library(
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
    srcs = glob(["*_test.go"]),
    deps = [
        ":helmdriver",
        "//internal/helm",
    ],
)
