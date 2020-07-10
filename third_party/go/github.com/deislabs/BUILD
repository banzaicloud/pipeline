package(default_visibility = ["PUBLIC"])

go_get(
    name = "oras",
    get = "github.com/deislabs/oras",
    install = [
        "pkg/auth",
        "pkg/auth/docker",
        "pkg/content",
        "pkg/context",
        "pkg/artifact",
        "pkg/oras",
    ],
    sum = "h1:If674KraJVpujYR00rzdi0QAmW4BxzMJPVAZJKuhQ0c=",
    version = "v0.8.1",
    deps = [
        "//third_party/go/github.com/containerd",
        "//third_party/go/github.com/docker",
        "//third_party/go/github.com/docker:cli",
        "//third_party/go/github.com/opencontainers:go-digest",
        "//third_party/go/github.com/opencontainers:image-spec",
        "//third_party/go/github.com/pkg:errors",
        "//third_party/go/github.com/sirupsen:logrus",
        "//third_party/go/golang.org/x:sync",
    ],
)
