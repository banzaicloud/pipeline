subinclude("///pleasings2//go:compat")

go_library(
    name = "http",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//pkg/problems"],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":http",
        "//pkg/problems",
    ],
)
