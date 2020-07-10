package(default_visibility = ["PUBLIC"])

go_get(
    name = "appkit",
    get = "github.com/sagikazarmark/appkit",
    install = [
        "errors",
        "transport/http",
        "endpoint",
        "run",
    ],
    sum = "h1:/Nv3L9SfbAa/2uM1+jWBPAnfo52Ei/ylwm/Ow1moHkA=",
    version = "v0.8.0",
    deps = [
        "//third_party/go/github.com/go-kit:kit",
        "//third_party/go/github.com/moogar0880:problems",
    ],
)

go_get(
    name = "kitx",
    get = "github.com/sagikazarmark/kitx",
    install = [
        "endpoint",
        "transport/http",
        "correlation",
        "transport",
    ],
    sum = "h1:19phhwBe4iVOy1Kam4dqGuxFT59/Wlp0xI239PhpaiQ=",
    version = "v0.12.0",
    deps = [
        "//third_party/go/github.com/go-kit:kit",
        "//third_party/go/github.com/moogar0880:problems",
        "//third_party/go/github.com/pkg:errors",
        "//third_party/go/google.golang.org:grpc",
    ],
)

go_get(
    name = "ocmux",
    get = "github.com/sagikazarmark/ocmux",
    install = [""],
    sum = "h1:YmxEdmIFCeM9NO99U3DBvLRq+cIy2gQJ4zDyifhOqWU=",
    version = "v0.2.0",
    deps = [
        "//third_party/go:go.opencensus.io",
        "//third_party/go/github.com/gorilla:mux",
    ],
)
