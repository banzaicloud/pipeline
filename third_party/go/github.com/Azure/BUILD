package(default_visibility = ["PUBLIC"])

go_get(
    name = "azure-pipeline-go",
    get = "github.com/Azure/azure-pipeline-go",
    install = ["pipeline"],
    sum = "h1:6oiIS9yaG6XCCzhgAgKFfIWyo4LLCiDhZot6ltoThhY=",
    version = "v0.2.2",
    deps = ["//third_party/go/github.com/mattn:go-ieproxy"],
)

go_get(
    name = "azure-sdk-for-go",
    get = "github.com/Azure/azure-sdk-for-go",
    install = [
        "version",
        "services/authorization/mgmt/2015-07-01/authorization",
        "services/compute/mgmt/2018-10-01/compute",
        "services/containerservice/mgmt/2020-02-01/containerservice",
        "services/network/mgmt/2018-10-01/network",
        "services/preview/monitor/mgmt/2018-09-01/insights",
        "services/resources/mgmt/2016-06-01/subscriptions",
        "services/resources/mgmt/2018-02-01/resources",
        "services/storage/mgmt/2017-10-01/storage",
        "profiles/latest/authorization/mgmt/authorization",
    ],
    sum = "h1:zeY2FNH9mauj53XcRJQ2JVSNH0M+eYxeWhep8nnhNoc=",
    version = "v39.2.0+incompatible",
    deps = [
        "//third_party/go/github.com/Azure/go-autorest:autorest",
        "//third_party/go/github.com/Azure/go-autorest:tracing",
        "//third_party/go/github.com/Azure/go-autorest/autorest:date",
        "//third_party/go/github.com/Azure/go-autorest/autorest:to",
        "//third_party/go/github.com/Azure/go-autorest/autorest:validation",
    ],
)

go_get(
    name = "azure-storage-blob-go",
    get = "github.com/Azure/azure-storage-blob-go",
    install = ["azblob"],
    sum = "h1:53qhf0Oxa0nOjgbDeeYPUeyiNmafAFEY95rZLK0Tj6o=",
    version = "v0.8.0",
    deps = [":azure-pipeline-go"],
)
