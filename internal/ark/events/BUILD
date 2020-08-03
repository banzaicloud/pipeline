subinclude("///pleasings2//go:compat")

go_library(
    name = "events",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/ark"],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":events",
        "//src/cluster",
    ],
)
