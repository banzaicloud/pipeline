subinclude("///pleasings2//go:compat")

filegroup(
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
    srcs = [":tokenadapter"],
    deps = ["//internal/app/pipeline/auth/token"],
)
