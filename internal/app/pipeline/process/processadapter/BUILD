subinclude("///pleasings2//go:compat")

filegroup(
    name = "processadapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/app/pipeline/process",
        "//pkg/gormhelper",
    ],
)
