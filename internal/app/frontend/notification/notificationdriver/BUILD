subinclude("///pleasings2//go:compat")

filegroup(
    name = "notificationdriver",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/app/frontend/notification",
        "//internal/platform/appkit/transport/http",
    ],
)

go_test(
    name = "test",
    srcs = [":notificationdriver"],
    deps = ["//internal/app/frontend/notification"],
)
