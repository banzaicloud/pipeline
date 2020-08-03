subinclude("///pleasings2//go:compat")

go_library(
    name = "action",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/global",
        "//internal/providers/alibaba/alibabaadapter",
        "//internal/secret/ssh/sshadapter",
        "//pkg/cluster/ack",
        "//pkg/errors",
        "//pkg/providers/alibaba",
        "//src/secret",
    ],
)
