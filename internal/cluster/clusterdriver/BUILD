subinclude("///pleasings2//go:compat")

go_library(
    name = "clusterdriver",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//.gen/pipeline/pipeline",
        "//internal/cluster",
        "//internal/platform/appkit/transport/http",
    ],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":clusterdriver",
        "//internal/cluster",
    ],
)
