package(default_visibility = ["PUBLIC"])

go_get(
    name = "gnostic",
    get = "github.com/googleapis/gnostic",
    install = [
        "extensions",
        "compiler",
        "OpenAPIv2",
    ],
    sum = "h1:WeAefnSUHlBb0iJKwxFDZdbfGwkd7xRNuV+IpXMJhYk=",
    version = "v0.3.1",
    deps = [
        "//third_party/go/github.com/golang:protobuf",
        "//third_party/go/gopkg.in:yaml.v2",
    ],
)
