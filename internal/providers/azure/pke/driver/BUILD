subinclude("///pleasings2//go:compat")

go_library(
    name = "driver",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster",
        "//internal/cluster/metrics",
        "//internal/global",
        "//internal/pke",
        "//internal/providers/azure/pke",
        "//internal/providers/azure/pke/driver/commoncluster",
        "//internal/providers/azure/pke/workflow",
        "//internal/secret/kubesecret",
        "//internal/secret/secrettype",
        "//internal/secret/ssh",
        "//internal/secret/sshsecret",
        "//pkg/cluster",
        "//pkg/cluster/pke",
        "//pkg/common",
        "//pkg/k8sclient",
        "//pkg/providers/azure",
        "//src/auth",
        "//src/cluster",
        "//src/secret",
    ],
)
