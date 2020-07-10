package(default_visibility = ["PUBLIC"])

go_get(
    name = "ark",
    get = "github.com/heptio/ark",
    install = [
        "pkg/apis/ark/v1",
        "pkg/generated/clientset/versioned/scheme",
        "pkg/cloudprovider",
    ],
    sum = "h1:LcyHwxub/geleuL6xEUk1xOBa6Fa8uT+/a7h2HEYBmc=",
    version = "v0.9.3",
    deps = [
        "//third_party/go/github.com/pkg:errors",
        "//third_party/go/github.com/sirupsen:logrus",
        "//third_party/go/k8s.io:api",
        "//third_party/go/k8s.io:apimachinery",
    ],
)
