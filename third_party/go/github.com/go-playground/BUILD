package(default_visibility = ["PUBLIC"])

go_get(
    name = "locales",
    get = "github.com/go-playground/locales",
    install = [
        "currency",
        "",
    ],
    sum = "h1:2FITxuFt/xuCNP1Acdhv62OzaCiviiE4kotfhkmOqEc=",
    version = "v0.12.1",
    deps = [],
)

go_get(
    name = "universal-translator",
    get = "github.com/go-playground/universal-translator",
    install = [""],
    sum = "h1:X++omBR/4cE2MNg91AoC3rmGrCjJ8eAeUP/K/EKx4DM=",
    version = "v0.16.0",
    deps = [":locales"],
)
