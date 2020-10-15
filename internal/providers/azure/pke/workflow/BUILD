subinclude("///pleasings2//go:compat")

go_library(
    name = "workflow",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster/clustersetup",
        "//internal/cluster/workflow",
        "//internal/pke",
        "//internal/pke/workflow",
        "//internal/providers/azure",
        "//internal/providers/azure/pke",
        "//internal/providers/pke/pkeworkflow",
        "//internal/providers/pke/pkeworkflow/pkeworkflowadapter",
        "//pkg/sdk/brn",
        "//pkg/cluster",
        "//pkg/cluster/pke",
        "//pkg/providers/azure",
        "//src/cluster",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [":workflow"],
)
