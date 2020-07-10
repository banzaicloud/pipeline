package(default_visibility = ["PUBLIC"])

go_get(
    name = "assetfs",
    get = "github.com/qor/assetfs",
    install = [""],
    sum = "h1:JRpyNNSRAkwNHd4WgyPcalTAhxOCh3eFNMoQkxWhjSw=",
    version = "v0.0.0-20170713023933-ff57fdc13a14",
    deps = [],
)

go_get(
    name = "auth",
    get = "github.com/qor/auth",
    install = [
        "claims",
        "auth_identity",
        "",
    ],
    replace = "github.com/banzaicloud/auth",
    sum = "h1:EUl1EMnJ2AfRGq6xn/SKdRY39ZULGyFj0XJDImv1aF4=",
    version = "v0.1.3",
    deps = [
        ":mailer",
        ":qor",
        ":redirect_back",
        ":render",
        ":responder",
        ":session",
        "//third_party/go/github.com/dgrijalva:jwt-go",
        "//third_party/go/github.com/jinzhu:copier",
        "//third_party/go/github.com/jinzhu:gorm",
    ],
)

go_get(
    name = "mailer",
    get = "github.com/qor/mailer",
    install = [
        "",
        "logger",
    ],
    sum = "h1:xD5EB1F5pmi1J48juHXdlOdLrhu/i7c7zvHdzgLMEJ4=",
    version = "v0.0.0-20170814094430-1e6ac7106955",
    deps = [
        ":assetfs",
        ":render",
        "//third_party/go/gopkg.in:gomail.v2",
    ],
)

go_get(
    name = "middlewares",
    get = "github.com/qor/middlewares",
    install = [""],
    sum = "h1:+WCc1IigwWpWBxMFsmLUsIF230TakGHstDajd8aKDAc=",
    version = "v0.0.0-20170822143614-781378b69454",
    deps = [],
)

go_get(
    name = "qor",
    get = "github.com/qor/qor",
    install = [
        "",
        "utils",
    ],
    sum = "h1:0z+LJ7Efz/a+SYR2Wr/CfSyLuFgzSVVFyu9hqGEY74Y=",
    version = "v0.0.0-20190319081902-186b0237364b",
    deps = [
        "//third_party/go/github.com/gosimple:slug",
        "//third_party/go/github.com/jinzhu:gorm",
        "//third_party/go/github.com/jinzhu:now",
        "//third_party/go/github.com/microcosm-cc:bluemonday",
    ],
)

go_get(
    name = "redirect_back",
    get = "github.com/qor/redirect_back",
    install = [""],
    sum = "h1:p8fFfW3hQzNMqXxr+LBrr82MIXoUi1HCXbeM8oCgfnw=",
    version = "v0.0.0-20170907030740-b4161ed6f848",
    deps = [
        ":middlewares",
        ":qor",
        ":session",
    ],
)

go_get(
    name = "render",
    get = "github.com/qor/render",
    install = [""],
    sum = "h1:4aq245m+d7ulq2cMQitfRP32qII5MwgE5tEe+UjfcEw=",
    version = "v0.0.0-20171201033449-63566e46f01b",
    deps = [
        ":assetfs",
        ":qor",
    ],
)

go_get(
    name = "responder",
    get = "github.com/qor/responder",
    install = [""],
    sum = "h1:g77aB9KGnuMC2kNSUWO2KN8k2RzgbI0HqOvWA+AlKX8=",
    version = "v0.0.0-20160314063933-ecae0be66c1a",
    deps = [],
)

go_get(
    name = "session",
    get = "github.com/qor/session",
    install = [
        "",
        "gorilla",
        "manager",
    ],
    sum = "h1:8l21EEdlZ9R0AA3FbeUAANc5NAx8Y3tn1VKbyAgjYlI=",
    version = "v0.0.0-20170907035918-8206b0adab70",
    deps = [
        ":middlewares",
        ":qor",
        "//third_party/go/github.com/gorilla:context",
        "//third_party/go/github.com/gorilla:sessions",
    ],
)
