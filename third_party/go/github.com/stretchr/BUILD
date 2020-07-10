package(default_visibility = ["PUBLIC"])

go_get(
    name = "objx",
    get = "github.com/stretchr/objx",
    install = [""],
    sum = "h1:Hbg2NidpLE8veEBkEZTL3CvlkUIVzuU9jDplZO54c48=",
    version = "v0.2.0",
    deps = [],
)

go_get(
    name = "testify",
    get = "github.com/stretchr/testify",
    install = [
        "assert",
        "mock",
        "require",
    ],
    sum = "h1:hDPOHmpOpP40lSULcqw7IrRb/u7w6RpDC9399XyoNd0=",
    version = "v1.6.1",
    deps = [
        ":objx",
        "//third_party/go/github.com/davecgh:go-spew",
        "//third_party/go/github.com/pmezard:go-difflib",
        "//third_party/go/gopkg.in:yaml.v3",
    ],
)
