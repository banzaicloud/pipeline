subinclude("///pleasings2//go:compat")

filegroup(
    name = "telemetry",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [],
)

go_test(
    name = "test",
    srcs = [":telemetry"],
    deps = [],
)
