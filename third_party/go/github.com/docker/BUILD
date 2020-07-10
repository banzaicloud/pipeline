package(default_visibility = ["PUBLIC"])

go_get(
    name = "cli",
    get = "github.com/docker/cli",
    install = [
        "cli/config/types",
        "cli/config/credentials",
        "cli/config/configfile",
        "cli/config",
    ],
    sum = "h1:FwssHbCDJD025h+BchanCwE1Q8fyMgqDr2mOQAWOLGw=",
    version = "v0.0.0-20200130152716-5d0cf8839492",
    deps = [
        ":docker",
        ":docker-credential-helpers",
        "//third_party/go/github.com/pkg:errors",
    ],
)

go_get(
    name = "distribution",
    get = "github.com/docker/distribution",
    install = [
        "digestset",
        "reference",
        "registry/api/errcode",
        "",
        "registry/api/v2",
        "registry/client/auth/challenge",
        "registry/client/transport",
        "metrics",
        "registry/storage/cache",
        "registry/storage/cache/memory",
        "registry/client",
        "registry/client/auth",
    ],
    sum = "h1:a5mlkVzth6W5A4fOsS3D2EO5BUmsJpcB+cRlLU7cSug=",
    version = "v2.7.1+incompatible",
    deps = [
        ":go-metrics",
        "//third_party/go/github.com/gorilla:mux",
        "//third_party/go/github.com/opencontainers:go-digest",
        "//third_party/go/github.com/opencontainers:image-spec",
    ],
)

go_get(
    name = "docker",
    get = "github.com/docker/docker",
    install = [
        "pkg/term",
        "pkg/homedir",
        "api/types/blkiodev",
        "api/types/mount",
        "api/types/strslice",
        "api/types/container",
        "api/types/versions",
        "api/types/filters",
        "errdefs",
        "api/types/network",
        "api/types/registry",
        "api/types/swarm/runtime",
        "api/types/swarm",
        "api/types",
        "pkg/ioutils",
        "pkg/jsonmessage",
        "pkg/stringid",
        "pkg/tarsum",
        "registry/resumable",
        "rootless",
        "registry",
    ],
    sum = "h1:KXS1Jg+ddGcWA8e1N7cupxaHHZhit5rB9tfDU+mfjyY=",
    version = "v1.4.2-0.20200203170920-46ec8731fbce",
    deps = [
        ":distribution",
        ":go-connections",
        ":go-units",
        "//third_party/go/github.com/containerd",
        "//third_party/go/github.com/gogo:protobuf",
        "//third_party/go/github.com/morikuni:aec",
        "//third_party/go/github.com/opencontainers:image-spec",
        "//third_party/go/github.com/pkg:errors",
        "//third_party/go/github.com/sirupsen:logrus",
        "//third_party/go/golang.org/x:sys",
        "//third_party/go/google.golang.org:grpc",
    ],
)

go_get(
    name = "docker-credential-helpers",
    get = "github.com/docker/docker-credential-helpers",
    install = [
        "credentials",
        "client",
    ],
    sum = "h1:zI2p9+1NQYdnG6sMU26EX4aVGlqbInSQxQXLvzJ4RPQ=",
    version = "v0.6.3",
    deps = [],
)

go_get(
    name = "go-connections",
    get = "github.com/docker/go-connections",
    install = [
        "nat",
        "tlsconfig",
    ],
    sum = "h1:El9xVISelRB7BuFusrZozjnkIM5YnzCViNKohAFqRJQ=",
    version = "v0.4.0",
    deps = ["//third_party/go/github.com/pkg:errors"],
)

go_get(
    name = "go-metrics",
    get = "github.com/docker/go-metrics",
    install = [""],
    sum = "h1:yWHOI+vFjEsAakUTSrtqc/SAHrhSkmn48pqjidZX3QA=",
    version = "v0.0.0-20180209012529-399ea8c73916",
    deps = ["//third_party/go/github.com/prometheus:client_golang"],
)

go_get(
    name = "go-units",
    get = "github.com/docker/go-units",
    install = [""],
    sum = "h1:3uh0PgVws3nIA0Q+MwDC8yjEPf9zjRfZZWXZYDct3Tw=",
    version = "v0.4.0",
    deps = [],
)

go_get(
    name = "spdystream",
    get = "github.com/docker/spdystream",
    install = [
        "spdy",
        "",
    ],
    sum = "h1:cenwrSVm+Z7QLSV/BsnenAOcDXdX4cMv4wP0B/5QbPg=",
    version = "v0.0.0-20160310174837-449fdfce4d96",
    deps = [],
)
