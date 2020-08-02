subinclude("///pleasings2//go:compat")

go_library(
    name = "app",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/app/pipeline/process",
        "//internal/app/pipeline/process/processadapter",
        "//internal/app/pipeline/process/processdriver",
        "//internal/platform/appkit/transport/http",
    ],
)
