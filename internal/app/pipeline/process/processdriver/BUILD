subinclude("///pleasings2//go:compat")

go_library(
    name = "processdriver",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//.gen/pipeline/pipeline",
        "//internal/app/pipeline/process",
        "//internal/platform/appkit/transport/http",
        "//src/auth",
    ],
)
