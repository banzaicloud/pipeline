subinclude("///pleasings2//go:compat")

go_library(
    name = "tokendriver",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/app/pipeline/auth/token",
        "//internal/platform/appkit/transport/http",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":tokendriver",
        "//internal/app/pipeline/auth/token",
    ],
)
