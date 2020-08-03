subinclude("///pleasings2//go:compat")

go_library(
    name = "integratedserviceadapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/common",
        "//internal/database/sql/json",
        "//internal/integratedservices",
        "//internal/integratedservices/integratedserviceadapter/workflow",
        "//pkg/cluster",
        "//src/cluster",
    ],
)
