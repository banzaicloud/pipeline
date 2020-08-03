subinclude("///pleasings2//go:compat")

go_library(
    name = "auditlogdriver",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//internal/platform/gin/auditlog",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":auditlogdriver",
        "//internal/common",
        "//internal/platform/gin/auditlog",
    ],
)
