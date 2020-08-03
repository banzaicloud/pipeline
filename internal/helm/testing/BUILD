subinclude("///pleasings2//go:compat")

go_library(
    name = "testing",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//internal/common/commonadapter",
        "//internal/global",
        "//internal/helm",
        "//internal/helm/helmadapter",
        "//internal/platform/log",
        "//internal/testing",
        "//src/secret",
    ],
)
