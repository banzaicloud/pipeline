package(default_visibility = ["PUBLIC"])

go_get(
    name = "optional",
    get = "cloud.google.com/go",
    install = ["internal/optional"],
    sum = "h1:PvKAVQWCtlGUSlZkGW3QLelKaWq7KYv/MW1EboG8bfM=",
    version = "v0.51.0",
    deps = [],
)

go_get(
    name = "trace",
    get = "cloud.google.com/go",
    install = ["internal/trace"],
    sum = "h1:PvKAVQWCtlGUSlZkGW3QLelKaWq7KYv/MW1EboG8bfM=",
    version = "v0.51.0",
    deps = [
        "//third_party/go:go.opencensus.io",
        "//third_party/go/google.golang.org:api",
        "//third_party/go/google.golang.org:genproto",
        "//third_party/go/google.golang.org:grpc",
    ],
)

go_get(
    name = "version",
    get = "cloud.google.com/go",
    install = ["internal/version"],
    sum = "h1:PvKAVQWCtlGUSlZkGW3QLelKaWq7KYv/MW1EboG8bfM=",
    version = "v0.51.0",
    deps = [],
)
