subinclude("///pleasings2//go:compat")

filegroup(
    name = "commoncluster",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/providers/vsphere/pke",
        "//internal/secret/secrettype",
        "//pkg/cluster",
        "//pkg/common",
        "//src/auth",
        "//src/secret",
    ],
)
