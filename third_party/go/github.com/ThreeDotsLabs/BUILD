package(default_visibility = ["PUBLIC"])

go_get(
    name = "watermill",
    get = "github.com/ThreeDotsLabs/watermill",
    install = [
        "",
        "internal",
        "pubsub/sync",
        "message",
        "components/cqrs",
        "message/router/middleware",
        "pubsub/gochannel",
    ],
    sum = "h1:RWVfySGHEaK4TZhr8L/rKkYkSbyzWRzE8ut7QP7esLY=",
    version = "v1.1.0",
    deps = [
        "//third_party/go/github.com/cenkalti/backoff:v3",
        "//third_party/go/github.com/gogo:protobuf",
        "//third_party/go/github.com/google:uuid",
        "//third_party/go/github.com/hashicorp:go-multierror",
        "//third_party/go/github.com/lithammer/shortuuid:v3",
        "//third_party/go/github.com/oklog:ulid",
        "//third_party/go/github.com/pkg:errors",
    ],
)
