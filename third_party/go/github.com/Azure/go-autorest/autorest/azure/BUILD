package(default_visibility = ["PUBLIC"])

go_get(
    name = "auth",
    get = "github.com/Azure/go-autorest/autorest/azure/auth",
    install = [""],
    sum = "h1:iM6UAvjR97ZIeR93qTcwpKNMpV+/FTWjwEbuPD495Tk=",
    version = "v0.4.2",
    deps = [
        ":cli",
        "//third_party/go/github.com/Azure/go-autorest:autorest",
        "//third_party/go/github.com/Azure/go-autorest/autorest:adal",
        "//third_party/go/github.com/dimchansky:utfbom",
        "//third_party/go/golang.org/x:crypto",
    ],
)

go_get(
    name = "cli",
    get = "github.com/Azure/go-autorest/autorest/azure/cli",
    install = [""],
    sum = "h1:LXl088ZQlP0SBppGFsRZonW6hSvwgL5gRByMbvUbx8U=",
    version = "v0.3.1",
    deps = [
        "//third_party/go/github.com/Azure/go-autorest/autorest:adal",
        "//third_party/go/github.com/Azure/go-autorest/autorest:date",
        "//third_party/go/github.com/dimchansky:utfbom",
        "//third_party/go/github.com/mitchellh:go-homedir",
    ],
)
