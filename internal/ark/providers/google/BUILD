subinclude("///pleasings2//go:compat")

go_library(
    name = "google",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/providers",
        "//internal/secret/secrettype",
        "//pkg/objectstore",
        "//pkg/providers/google/objectstore",
        "//src/secret",
    ],
)
