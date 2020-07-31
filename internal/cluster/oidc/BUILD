subinclude("///pleasings2//go:compat")

filegroup(
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
    srcs = [":oidc"],
    deps = [
        "//internal/cluster/auth",
        "//src/auth",
    ],
)
