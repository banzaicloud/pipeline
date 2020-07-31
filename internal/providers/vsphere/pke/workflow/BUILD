subinclude("///pleasings2//go:compat")

filegroup(
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
        "//internal/providers/pke/pkeworkflow",
        "//internal/providers/pke/pkeworkflow/pkeworkflowadapter",
        "//internal/providers/vsphere/pke",
        "//internal/secret/secrettype",
        "//pkg/brn",
        "//pkg/cluster",
        "//pkg/k8sclient",
        "//src/cluster",
    ],
)

go_test(
    name = "test",
    srcs = [":workflow"],
    deps = [],
)
