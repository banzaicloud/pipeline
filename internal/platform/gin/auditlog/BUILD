subinclude("///pleasings2//go:compat")

go_library(
    name = "auditlog",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//internal/platform/gin/correlationid",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":auditlog",
        "//internal/platform/gin/correlationid",
    ],
)
