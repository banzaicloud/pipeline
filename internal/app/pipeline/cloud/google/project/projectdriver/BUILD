subinclude("///pleasings2//go:compat")

filegroup(
    name = "projectdriver",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/app/pipeline/cloud/google/project",
        "//internal/platform/appkit/transport/http",
    ],
)

go_test(
    name = "test",
    srcs = [":projectdriver"],
    deps = ["//internal/app/pipeline/cloud/google/project"],
)
