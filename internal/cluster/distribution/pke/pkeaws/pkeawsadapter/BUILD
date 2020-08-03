subinclude("///pleasings2//go:compat")

go_library(
    name = "pkeawsadapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//.gen/cloudinfo",
        "//internal/cluster/distribution/pke/pkeaws",
    ],
)
