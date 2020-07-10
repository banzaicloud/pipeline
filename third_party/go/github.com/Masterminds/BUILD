package(default_visibility = ["PUBLIC"])

go_get(
    name = "goutils",
    get = "github.com/Masterminds/goutils",
    install = [""],
    sum = "h1:zukEsf/1JZwCMgHiK3GZftabmxiCw4apj3a28RPBiVg=",
    version = "v1.1.0",
    deps = [],
)

go_get(
    name = "semver",
    get = "github.com/Masterminds/semver",
    install = [""],
    sum = "h1:H65muMkzWKEuNDnfl9d70GUjFniHKHRbFPGBuZ3QEww=",
    version = "v1.5.0",
    deps = [],
)

go_get(
    name = "sprig",
    get = "github.com/Masterminds/sprig",
    install = [""],
    sum = "h1:z4yfnGrZ7netVz+0EDJ0Wi+5VZCSYp4Z0m2dk6cEM60=",
    version = "v2.22.0+incompatible",
    deps = [
        ":goutils",
        ":semver",
        "//third_party/go/github.com/google:uuid",
        "//third_party/go/github.com/huandu:xstrings",
        "//third_party/go/github.com/imdario:mergo",
        "//third_party/go/github.com/mitchellh:copystructure",
        "//third_party/go/golang.org/x:crypto",
    ],
)
