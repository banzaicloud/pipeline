package(default_visibility = ["PUBLIC"])

go_get(
    name = "autorest",
    get = "github.com/Azure/go-autorest/autorest",
    install = [
        "",
        "azure",
    ],
    sum = "h1:OZEIaBbMdUE/Js+BQKlpO81XlISgipr6yDJ+PSwsgi4=",
    version = "v0.9.3",
    deps = [
        ":logger",
        ":tracing",
        "//third_party/go/github.com/Azure/go-autorest/autorest:adal",
    ],
)

go_get(
    name = "logger",
    get = "github.com/Azure/go-autorest/logger",
    install = [""],
    sum = "h1:ruG4BSDXONFRrZZJ2GUXDiUyVpayPmb1GnWeHDdaNKY=",
    version = "v0.1.0",
    deps = [],
)

go_get(
    name = "tracing",
    get = "github.com/Azure/go-autorest/tracing",
    install = [""],
    sum = "h1:TRn4WjSnkcSy5AEG3pnbtFSwNtwzjr4VYyQflFE619k=",
    version = "v0.5.0",
    deps = [],
)
