package(default_visibility = ["PUBLIC"])

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

go_get(
    name = "iam",
    get = "cloud.google.com/go",
    install = ["iam"],
    sum = "h1:PvKAVQWCtlGUSlZkGW3QLelKaWq7KYv/MW1EboG8bfM=",
    version = "v0.51.0",
    deps = [
        "//third_party/go/github.com/googleapis/gax-go:v2",
        "//third_party/go/google.golang.org:genproto",
        "//third_party/go/google.golang.org:grpc",
    ],
)

go_get(
    name = "internal",
    get = "cloud.google.com/go",
    install = ["internal"],
    sum = "h1:PvKAVQWCtlGUSlZkGW3QLelKaWq7KYv/MW1EboG8bfM=",
    version = "v0.51.0",
    deps = [
        "//third_party/go/github.com/googleapis/gax-go:v2",
        "//third_party/go/google.golang.org:api",
        "//third_party/go/google.golang.org:grpc",
    ],
)

go_get(
    name = "errorreporting",
    get = "cloud.google.com/go",
    install = ["errorreporting"],
    sum = "h1:PvKAVQWCtlGUSlZkGW3QLelKaWq7KYv/MW1EboG8bfM=",
    version = "v0.51.0",
    deps = [
        "//third_party/go/cloud.google.com/go/errorreporting:apiv1beta1",
        "//third_party/go/cloud.google.com/go/internal:version",
        "//third_party/go/github.com/golang:protobuf",
        "//third_party/go/github.com/googleapis/gax-go:v2",
        "//third_party/go/google.golang.org:api",
        "//third_party/go/google.golang.org:genproto",
    ],
)

go_get(
    name = "storage",
    get = "cloud.google.com/go/storage",
    install = [""],
    sum = "h1:VV2nUM3wwLLGh9lSABFgZMjInyUbJeaRSE64WuAIQ+4=",
    version = "v1.0.0",
    deps = [
        ":iam",
        ":internal",
        "//third_party/go/cloud.google.com/go/internal:optional",
        "//third_party/go/cloud.google.com/go/internal:trace",
        "//third_party/go/cloud.google.com/go/internal:version",
        "//third_party/go/github.com/googleapis/gax-go:v2",
        "//third_party/go/google.golang.org:api",
        "//third_party/go/google.golang.org:genproto",
    ],
)
