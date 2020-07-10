package(default_visibility = ["PUBLIC"])

go_get(
    name = "sdk",
    get = "github.com/banzaicloud/logging-operator/pkg/sdk",
    install = [
        "model/types",
        "model/filter",
        "model/common",
        "model/output",
        "api/v1beta1",
    ],
    sum = "h1:96DCkcHkbDgaCLpvzrdyQ/lzzntOSp/Wklg6MTYI5GQ=",
    version = "v0.3.15",
    deps = [
        "//third_party/go/emperror.dev:errors",
        "//third_party/go/github.com/banzaicloud:operator-tools",
        "//third_party/go/github.com/spf13:cast",
        "//third_party/go/k8s.io:api",
        "//third_party/go/k8s.io:apimachinery",
        "//third_party/go/sigs.k8s.io:controller-runtime",
    ],
)
