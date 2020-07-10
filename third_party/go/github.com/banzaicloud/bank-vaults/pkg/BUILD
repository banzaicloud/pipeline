package(default_visibility = ["PUBLIC"])

go_get(
    name = "sdk",
    get = "github.com/banzaicloud/bank-vaults/pkg/sdk",
    install = [
        "vault",
        "auth",
        "db",
        "tls",
    ],
    sum = "h1:YBo1RVbiq/o14Z8x2K/hH0qF2iFhe1OBphoPwLggTa4=",
    version = "v0.3.1",
    deps = [
        "//third_party/go/emperror.dev:errors",
        "//third_party/go/github.com/fsnotify",
        "//third_party/go/github.com/hashicorp:hcl",
        "//third_party/go/github.com/hashicorp/vault:api",
        "//third_party/go/github.com/hashicorp/vault:sdk",
        "//third_party/go/github.com/json-iterator:go",
        "//third_party/go/github.com/mitchellh:mapstructure",
        "//third_party/go/github.com/sirupsen:logrus",
        "//third_party/go/github.com/spf13:cast",
        "//third_party/go/github.com/spf13:viper",
        "//third_party/go/k8s.io:api",
        "//third_party/go/k8s.io:client-go",
        "//third_party/go/sigs.k8s.io:controller-runtime",
    ],
)
