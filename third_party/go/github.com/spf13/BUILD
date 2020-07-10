package(default_visibility = ["PUBLIC"])

go_get(
    name = "afero",
    get = "github.com/spf13/afero",
    install = [
        "mem",
        "",
    ],
    sum = "h1:5jhuqJyZCZf2JRofRvN/nIFgIWNzPa3/Vz8mYylgbWc=",
    version = "v1.2.2",
    deps = ["//third_party/go/golang.org/x:text"],
)

go_get(
    name = "cast",
    get = "github.com/spf13/cast",
    install = [""],
    sum = "h1:nFm6S0SMdyzrzcmThSipiEubIDy8WEXKNZ0UOgiRpng=",
    version = "v1.3.1",
    deps = [],
)

go_get(
    name = "cobra",
    get = "github.com/spf13/cobra",
    install = [""],
    sum = "h1:6m/oheQuQ13N9ks4hubMG6BnvwOeaJrqSPLahSnczz8=",
    version = "v1.0.0",
    deps = [":pflag"],
)

go_get(
    name = "jwalterweatherman",
    get = "github.com/spf13/jwalterweatherman",
    install = [""],
    sum = "h1:ue6voC5bR5F8YxI5S67j9i582FU4Qvo2bmqnqMYADFk=",
    version = "v1.1.0",
    deps = [],
)

go_get(
    name = "pflag",
    get = "github.com/spf13/pflag",
    install = [""],
    sum = "h1:iy+VFUOCP1a+8yFto/drg2CJ5u0yRoB7fZw3DKv/JXA=",
    version = "v1.0.5",
    deps = [],
)

go_get(
    name = "viper",
    get = "github.com/spf13/viper",
    install = [""],
    sum = "h1:xVKxvI7ouOI5I+U9s2eeiUfMaWBVoXA3AWskkrqK0VM=",
    version = "v1.7.0",
    deps = [
        ":afero",
        ":cast",
        ":jwalterweatherman",
        ":pflag",
        "//third_party/go/github.com/fsnotify",
        "//third_party/go/github.com/hashicorp:hcl",
        "//third_party/go/github.com/magiconair:properties",
        "//third_party/go/github.com/mitchellh:mapstructure",
        "//third_party/go/github.com/pelletier:go-toml",
        "//third_party/go/github.com/subosito:gotenv",
        "//third_party/go/gopkg.in:ini.v1",
        "//third_party/go/gopkg.in:yaml.v2",
    ],
)
