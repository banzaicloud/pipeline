subinclude("///pleasings2//go:compat")

go_library(
    name = "oidc",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster/auth",
        "//src/auth",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":oidc",
        "//internal/cluster/auth",
        "//src/auth",
    ],
)
