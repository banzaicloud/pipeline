go_library(
    name = "cloudinfo",
    srcs = glob(
        ["*.go"],
        exclude = ["*_test.go"],
    ),
    visibility = ["PUBLIC"],
    deps = [
        "//third_party/go/github.com/antihax/optional",
        "//third_party/go/golang.org/x/oauth2",
    ],
)
