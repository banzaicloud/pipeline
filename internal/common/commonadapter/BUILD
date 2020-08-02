subinclude("///pleasings2//go:compat")

go_library(
    name = "commonadapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//pkg/brn",
        "//src/secret",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":commonadapter",
        "//internal/common",
        "//internal/secret/secrettype",
        "//src/secret",
    ],
)
