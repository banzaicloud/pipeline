subinclude("///pleasings2//go:compat")

go_library(
    name = "cloudinfo",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = ["//.gen/cloudinfo"],
)
