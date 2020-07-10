package(default_visibility = ["PUBLIC"])

go_get(
    name = "gojsonpointer",
    get = "github.com/xeipuuv/gojsonpointer",
    install = [""],
    sum = "h1:J9EGpcZtP0E/raorCMxlFGSTBrsSlaDGf3jU/qvAE2c=",
    version = "v0.0.0-20180127040702-4e3ac2762d5f",
    deps = [],
)

go_get(
    name = "gojsonreference",
    get = "github.com/xeipuuv/gojsonreference",
    install = [""],
    sum = "h1:EzJWgHovont7NscjpAxXsDA8S8BMYve8Y5+7cuRE7R0=",
    version = "v0.0.0-20180127040603-bd5ef7bd5415",
    deps = [":gojsonpointer"],
)

go_get(
    name = "gojsonschema",
    get = "github.com/xeipuuv/gojsonschema",
    install = [""],
    sum = "h1:ngVtJC9TY/lg0AA/1k48FYhBrhRoFlEmWzsehpNAaZg=",
    version = "v1.1.0",
    deps = [":gojsonreference"],
)
