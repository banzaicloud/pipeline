package(default_visibility = ["PUBLIC"])

go_get(
    name = "jsonpointer",
    get = "github.com/go-openapi/jsonpointer",
    install = [""],
    sum = "h1:gihV7YNZK1iK6Tgwwsxo2rJbD1GTbdm72325Bq8FI3w=",
    version = "v0.19.3",
    deps = [":swag"],
)

go_get(
    name = "jsonreference",
    get = "github.com/go-openapi/jsonreference",
    install = [""],
    sum = "h1:5cxNfTy0UVC3X8JL5ymxzyoUZmo8iZb+jeTWn7tUa8o=",
    version = "v0.19.3",
    deps = [
        ":jsonpointer",
        "//third_party/go/github.com/PuerkitoBio:purell",
    ],
)

go_get(
    name = "spec",
    get = "github.com/go-openapi/spec",
    install = [""],
    sum = "h1:0XRyw8kguri6Yw4SxhsQA/atC88yqrk0+G4YhI2wabc=",
    version = "v0.19.3",
    deps = [
        ":jsonpointer",
        ":jsonreference",
        ":swag",
    ],
)

go_get(
    name = "swag",
    get = "github.com/go-openapi/swag",
    install = [""],
    sum = "h1:lTz6Ys4CmqqCQmZPBlbQENR1/GucA2bzYTE12Pw4tFY=",
    version = "v0.19.5",
    deps = [
        "//third_party/go/github.com/mailru:easyjson",
        "//third_party/go/gopkg.in:yaml.v2",
    ],
)
