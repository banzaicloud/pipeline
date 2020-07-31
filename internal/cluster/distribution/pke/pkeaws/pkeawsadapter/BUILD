subinclude("///pleasings2//go:compat")

filegroup(
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
