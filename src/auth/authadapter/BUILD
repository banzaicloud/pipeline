subinclude("///pleasings2//go:compat")

go_library(
    name = "authadapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//src/auth"],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":authadapter",
        "//internal/common",
        "//src/auth",
    ],
)
