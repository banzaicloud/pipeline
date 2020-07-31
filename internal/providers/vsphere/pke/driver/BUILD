subinclude("///pleasings2//go:compat")

filegroup(
    name = "driver",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/cluster",
        "//internal/cluster/metrics",
        "//internal/common",
        "//internal/global",
        "//internal/pke",
        "//internal/providers/vsphere/pke",
        "//internal/providers/vsphere/pke/driver/commoncluster",
        "//internal/providers/vsphere/pke/workflow",
        "//internal/secret/secrettype",
        "//internal/secret/ssh",
        "//internal/secret/sshsecret",
        "//pkg/cluster",
        "//pkg/cluster/pke",
        "//pkg/common",
        "//src/auth",
        "//src/cluster",
        "//src/secret",
        "//src/utils",
    ],
)
