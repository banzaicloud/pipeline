subinclude("///pleasings2//go:compat")

go_library(
    name = "frontend",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/app/frontend/notification",
        "//internal/app/frontend/notification/notificationadapter",
        "//internal/app/frontend/notification/notificationdriver",
        "//internal/common",
        "//internal/platform/appkit/transport/http",
    ],
)
