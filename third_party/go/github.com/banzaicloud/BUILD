package(default_visibility = ["PUBLIC"])

go_get(
    name = "anchore-image-validator",
    get = "github.com/banzaicloud/anchore-image-validator",
    install = ["pkg/apis/security/v1alpha1"],
    sum = "h1:wlvTg+btN2gyZrKDOoCpAz+tJLnc072UbTfFTtEdnZk=",
    version = "v0.0.0-20190823121528-918b9fa6af62",
    deps = ["//third_party/go/k8s.io:apimachinery"],
)

go_get(
    name = "gin-utilz",
    get = "github.com/banzaicloud/gin-utilz",
    install = ["auth"],
    sum = "h1:j6wLdZSlw36g63yklk11ihnIKqoU2zFfXIcaL1dumuM=",
    version = "v0.2.0",
    deps = [
        "//third_party/go/emperror.dev:errors",
        "//third_party/go/github.com/dgrijalva:jwt-go",
        "//third_party/go/github.com/gin-gonic:gin",
    ],
)

go_get(
    name = "go-gin-prometheus",
    get = "github.com/banzaicloud/go-gin-prometheus",
    install = [""],
    sum = "h1:zetacN1k3svVc7DtzGfpHA49uO0G8cwsiP5qSuUUQKs=",
    version = "v0.1.0",
    deps = [
        "//third_party/go/github.com/gin-gonic:gin",
        "//third_party/go/github.com/prometheus:client_golang",
        "//third_party/go/github.com/sirupsen:logrus",
    ],
)

go_get(
    name = "istio-operator",
    get = "github.com/banzaicloud/istio-operator",
    install = [
        "pkg/util",
        "pkg/apis/istio/v1beta1",
    ],
    sum = "h1:nz020decQ0q8AkYzrnc/NZTbcuzw9B8N+s7LW5DA+3E=",
    version = "v0.0.0-20200330114955-d15bdd228ae4",
    deps = [
        "//third_party/go/k8s.io:api",
        "//third_party/go/k8s.io:apimachinery",
        "//third_party/go/sigs.k8s.io:controller-runtime",
    ],
)

go_get(
    name = "k8s-objectmatcher",
    get = "github.com/banzaicloud/k8s-objectmatcher",
    install = ["patch"],
    sum = "h1:HhXkOWg4xmAW203p3G4Av9ylUfkUJF1+8MK90XQAaJw=",
    version = "v1.3.2",
    deps = [
        "//third_party/go/github.com/goph:emperror",
        "//third_party/go/github.com/json-iterator:go",
        "//third_party/go/github.com/pkg:errors",
        "//third_party/go/k8s.io:api",
        "//third_party/go/k8s.io:apimachinery",
    ],
)

go_get(
    name = "logrus-runtime-formatter",
    get = "github.com/banzaicloud/logrus-runtime-formatter",
    install = [""],
    sum = "h1:WeGZeO6I+lI1IG2vm1iXTeMrHLcMUjfoYm+rubBM5c4=",
    version = "v0.0.0-20180617171254-12df4a18567f",
    deps = ["//third_party/go/github.com/sirupsen:logrus"],
)

go_get(
    name = "operator-tools",
    get = "github.com/banzaicloud/operator-tools",
    install = [
        "pkg/secret",
        "pkg/utils",
        "pkg/volume",
    ],
    sum = "h1:Nv+nhPs2EjO5L4YjgSsuMVJKl81xMZh/T3zGNBKahFE=",
    version = "v0.10.2",
    deps = [
        "//third_party/go/emperror.dev:errors",
        "//third_party/go/github.com/go-logr:logr",
        "//third_party/go/github.com/iancoleman:orderedmap",
        "//third_party/go/github.com/spf13:cast",
        "//third_party/go/k8s.io:api",
        "//third_party/go/k8s.io:apimachinery",
        "//third_party/go/sigs.k8s.io:controller-runtime",
    ],
)
