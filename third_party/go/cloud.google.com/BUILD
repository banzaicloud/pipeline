package(default_visibility = ["PUBLIC"])

go_get_download(
    name = "go",
    tag = "get",
    get = "cloud.google.com/go",
    sum = "h1:PvKAVQWCtlGUSlZkGW3QLelKaWq7KYv/MW1EboG8bfM=",
    version = "v0.51.0",
)

go_get_build(
    name = "iam2",
    get = "cloud.google.com/go",
    install = ["iam"],
    srcs = ["//third_party/go/cloud.google.com:_go#get"],
    deps = [
        "//third_party/go/github.com/googleapis/gax-go:v2",
        "//third_party/go/google.golang.org:genproto",
        "//third_party/go/google.golang.org:grpc",
    ],
)

go_get_build(
    name = "internal",
    get = "cloud.google.com/go",
    install = ["internal"],
    srcs = ["//third_party/go/cloud.google.com:_go#get"],
    deps = [
        #"//third_party/go/github.com/googleapis/gax-go:v2",
        #"//third_party/go/google.golang.org:api",
        #"//third_party/go/google.golang.org:grpc",
    ],
)
