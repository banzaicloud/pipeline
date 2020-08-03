subinclude("///pleasings2//go:compat")

go_library(
    name = "notificationadapter",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//internal/app/frontend/notification"],
)

go_test(
    name = "test",
    srcs = glob(["*_test.go"]),
    deps = [
        ":notificationadapter",
        "//internal/app/frontend/notification",
    ],
)

go_test(
    name = "integration_test",
    srcs = glob(["*_test.go"]),
    flags = "-test.run ^TestIntegration$",
    labels = ["integration"],
    deps = [
        ":notificationadapter",
        "//internal/app/frontend/notification",
    ],
)
