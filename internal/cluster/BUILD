subinclude("///pleasings2//go:compat")

go_library(
    name = "cluster",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//pkg/brn",
        "//pkg/cloud",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":cluster",
        "//pkg/brn",
        "//pkg/cloud",
    ],
)
