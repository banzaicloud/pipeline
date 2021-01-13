subinclude("///pleasings2//go:compat")

go_library(
    name = "awsworkflow",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster",
        ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":awsworkflow",
    ],
)
