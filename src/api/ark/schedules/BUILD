subinclude("///pleasings2//go:compat")

go_library(
    name = "schedules",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/ark/api",
        "//internal/global",
        "//internal/platform/gin/correlationid",
        "//src/api/ark/common",
    ],
)
