subinclude("///pleasings2//go:compat")

go_library(
    name = "alibaba",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/network",
        "//internal/objectstore",
        "//internal/secret/secrettype",
        "//pkg/objectstore",
        "//pkg/providers",
        "//pkg/providers/alibaba",
        "//pkg/providers/alibaba/objectstore",
        "//src/auth",
        "//src/secret",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [":alibaba"],
)
