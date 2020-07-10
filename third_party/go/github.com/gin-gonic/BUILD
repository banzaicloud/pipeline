package(default_visibility = ["PUBLIC"])

go_get(
    name = "gin",
    get = "github.com/gin-gonic/gin",
    install = [
        "internal/json",
        "binding",
        "render",
        "",
    ],
    sum = "h1:fi+bqFAx/oLK54somfCtEZs9HeH1LHVoEPUgARpTqyc=",
    version = "v1.5.0",
    deps = [
        "//third_party/go/github.com/gin-contrib:sse",
        "//third_party/go/github.com/golang:protobuf",
        "//third_party/go/github.com/mattn:go-isatty",
        "//third_party/go/github.com/ugorji/go:codec",
        "//third_party/go/gopkg.in:yaml.v2",
        "//third_party/go/gopkg.in/go-playground:validator.v9",
    ],
)
