subinclude("///pleasings2//go:compat")

go_library(
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
    srcs = glob(["*_test.go"]),
    deps = [
        ":projectdriver",
        "//internal/app/pipeline/cloud/google/project",
    ],
)
