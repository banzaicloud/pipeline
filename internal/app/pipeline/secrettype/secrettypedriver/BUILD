subinclude("///pleasings2//go:compat")

go_library(
    name = "secrettypedriver",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/app/pipeline/secrettype",
        "//internal/platform/appkit/transport/http",
    ],
)
