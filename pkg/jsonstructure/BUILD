subinclude("///pleasings2//go:compat")

filegroup(
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
    srcs = [":jsonstructure"],
    deps = ["//pkg/any"],
)
