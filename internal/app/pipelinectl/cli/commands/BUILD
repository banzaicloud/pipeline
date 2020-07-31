subinclude("///pleasings2//go:compat")

filegroup(
    name = "commands",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/app/pipelinectl/cli/commands/drain",
        "//internal/app/pipelinectl/cli/commands/telemetry",
    ],
)
