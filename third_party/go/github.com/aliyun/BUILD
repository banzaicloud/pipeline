package(default_visibility = ["PUBLIC"])

go_get(
    name = "alibaba-cloud-sdk-go",
    get = "github.com/aliyun/alibaba-cloud-sdk-go",
    install = [
        "sdk/auth/credentials",
        "sdk/utils",
        "sdk/errors",
        "sdk/requests",
        "sdk/responses",
        "sdk/auth/signers",
        "sdk/auth",
        "sdk/auth/credentials/provider",
        "sdk/endpoints",
        "sdk",
        "services/vpc",
        "services/cs",
        "services/ecs",
        "services/ess",
    ],
    sum = "h1:MZrGWhrO7yXMOv+Ul0V/yXzxQ3Pso6WGy+SNP2102+c=",
    version = "v1.60.327",
    deps = [
        "//third_party/go/github.com/jmespath:go-jmespath",
        "//third_party/go/github.com/json-iterator:go",
        "//third_party/go/gopkg.in:ini.v1",
    ],
)

go_get(
    name = "aliyun-oss-go-sdk",
    get = "github.com/aliyun/aliyun-oss-go-sdk",
    install = ["oss"],
    sum = "h1:A3oZlWPD/Poa19FvNbw+Zu4yKAurDBTjlRDilYGBiS4=",
    version = "v2.0.5+incompatible",
    deps = ["//third_party/go/golang.org/x:time"],
)
