subinclude("///pleasings2//go:compat")

filegroup(
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
