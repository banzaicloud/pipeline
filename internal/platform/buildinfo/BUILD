subinclude("///pleasings2//go:compat")

go_library(
    name = "buildinfo",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//.gen/pipeline/pipeline",
        "//internal/global",
    ],
)
