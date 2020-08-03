subinclude("///pleasings2//go:compat")

go_library(
    name = "pke",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/platform/gin/utils",
        "//pkg/common",
        "//src/api/common",
        "//src/cluster",
    ],
)
