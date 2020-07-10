package(default_visibility = ["PUBLIC"])

go_get(
    name = "mapdecode",
    get = "github.com/uber-go/mapdecode",
    install = [
        "internal/mapstructure",
        "",
    ],
    sum = "h1:euUEFM9KnuCa1OBixz1xM+FIXmpixyay5DLymceOVrU=",
    version = "v1.0.0",
    deps = ["//third_party/go/go.uber.org:multierr"],
)

go_get(
    name = "tally",
    get = "github.com/uber-go/tally",
    install = [""],
    sum = "h1:9hLSgNBP28CjIaDmAuRTq9qV+UZY+9PcvAkXO4nNMwg=",
    version = "v3.3.15+incompatible",
    deps = [],
)
