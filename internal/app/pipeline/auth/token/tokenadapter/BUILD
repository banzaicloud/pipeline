subinclude("///pleasings2//go:compat")

go_library(
    name = "tokenadapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/app/pipeline/auth/token"],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":tokenadapter",
        "//internal/app/pipeline/auth/token",
    ],
)
