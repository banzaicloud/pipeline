subinclude("///pleasings2//go:compat")

go_library(
    name = "jsonstructure",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//pkg/any",
        "//pkg/mirror",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":jsonstructure",
        "//pkg/any",
    ],
)
