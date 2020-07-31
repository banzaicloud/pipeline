subinclude("///pleasings2//go:compat")

filegroup(
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
    srcs = [":notificationadapter"],
    deps = ["//internal/app/frontend/notification"],
)

go_test(
    name = "integration_test",
    srcs = [":notificationadapter"],
    flags = "-test.run ^TestIntegration$",
    labels = ["integration"],
    deps = ["//internal/app/frontend/notification"],
)
