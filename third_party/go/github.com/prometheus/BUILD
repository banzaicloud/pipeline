package(default_visibility = ["PUBLIC"])

go_get(
    name = "client_golang",
    get = "github.com/prometheus/client_golang",
    install = [
        "prometheus/internal",
        "prometheus",
        "prometheus/promhttp",
        "api",
        "api/prometheus/v1",
    ],
    sum = "h1:FFSuS004yOQEtDdTq+TAOLP5xUq63KqAFYyOi8zA+Y8=",
    version = "v1.4.1",
    deps = [
        ":client_model",
        ":common",
        ":procfs",
        "//third_party/go/github.com/beorn7:perks",
        "//third_party/go/github.com/cespare/xxhash:v2",
        "//third_party/go/github.com/golang:protobuf",
        "//third_party/go/github.com/json-iterator:go",
    ],
)

go_get(
    name = "client_model",
    get = "github.com/prometheus/client_model",
    install = ["go"],
    sum = "h1:uq5h0d+GuxiXLJLNABMgp2qUWDPiLvgCzz2dUR+/W/M=",
    version = "v0.2.0",
    deps = ["//third_party/go/github.com/golang:protobuf"],
)

go_get(
    name = "common",
    get = "github.com/prometheus/common",
    install = [
        "internal/bitbucket.org/ww/goautoneg",
        "model",
        "expfmt",
    ],
    sum = "h1:KOMtN28tlbam3/7ZKEYKHhKoJZYYj3gMH4uc62x7X7U=",
    version = "v0.9.1",
    deps = [
        ":client_model",
        "//third_party/go/github.com/golang:protobuf",
        "//third_party/go/github.com/matttproud:golang_protobuf_extensions",
    ],
)

go_get(
    name = "procfs",
    get = "github.com/prometheus/procfs",
    install = [
        "internal/fs",
        "internal/util",
        "",
    ],
    sum = "h1:DksSrntiTPE63NQuxGcFa1OS/odKfwJu3PJHrhKAy7Q=",
    version = "v0.0.9",
    deps = [],
)
