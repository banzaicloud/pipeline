subinclude("///pleasings2//go:compat")

go_library(
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
    srcs = glob(["*_test.go"]),
    deps = [
        ":notificationdriver",
        "//internal/app/frontend/notification",
    ],
)
