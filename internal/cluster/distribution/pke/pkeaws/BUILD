subinclude("///pleasings2//go:compat")

go_library(
    name = "pkeaws",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/common"],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [":pkeaws"],
)
