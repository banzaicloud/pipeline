subinclude("///pleasings2//go:compat")

filegroup(
    name = "cluster",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//.gen/pipeline/pipeline",
        "//internal/pke",
        "//internal/providers/azure/pke",
        "//internal/providers/azure/pke/driver",
        "//internal/providers/vsphere/pke",
        "//internal/providers/vsphere/pke/driver",
        "//pkg/cluster",
        "//src/secret",
    ],
)

go_test(
    name = "test",
    srcs = [":cluster"],
    deps = [
        "//.gen/pipeline/pipeline",
        "//internal/pke",
        "//internal/providers/azure/pke",
        "//internal/providers/azure/pke/driver",
        "//pkg/cluster",
    ],
)
