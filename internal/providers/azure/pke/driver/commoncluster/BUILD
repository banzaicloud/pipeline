subinclude("///pleasings2//go:compat")

go_library(
    name = "commoncluster",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/providers/azure/pke",
        "//internal/secret/secrettype",
        "//pkg/cluster",
        "//pkg/common",
        "//src/auth",
        "//src/secret",
    ],
)
