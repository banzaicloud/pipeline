subinclude("///pleasings2//go:compat")

filegroup(
    name = "capdriver",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/app/pipeline/cap"],
)
