package(default_visibility = ["PUBLIC"])

go_get(
    name = "api",
    get = "github.com/hashicorp/vault/api",
    install = [""],
    sum = "h1:j08Or/wryXT4AcHj1oCbMd7IijXcKzYUGw59LGu9onU=",
    version = "v1.0.4",
    deps = [
        ":sdk",
        "//third_party/go/github.com/hashicorp:errwrap",
        "//third_party/go/github.com/hashicorp:go-cleanhttp",
        "//third_party/go/github.com/hashicorp:go-multierror",
        "//third_party/go/github.com/hashicorp:go-retryablehttp",
        "//third_party/go/github.com/hashicorp:go-rootcerts",
        "//third_party/go/github.com/hashicorp:hcl",
        "//third_party/go/github.com/mitchellh:mapstructure",
        "//third_party/go/golang.org/x:net",
        "//third_party/go/golang.org/x:time",
        "//third_party/go/gopkg.in/square:go-jose.v2",
    ],
)

go_get(
    name = "sdk",
    get = "github.com/hashicorp/vault/sdk",
    install = [
        "helper/consts",
        "helper/hclutil",
        "helper/compressutil",
        "helper/jsonutil",
        "helper/strutil",
        "helper/parseutil",
    ],
    sum = "h1:mOEPeOhT7jl0J4AMl1E705+BcmeRs1VmKNb9F0sMLy8=",
    version = "v0.1.13",
    deps = [
        "//third_party/go/github.com/golang:snappy",
        "//third_party/go/github.com/hashicorp:errwrap",
        "//third_party/go/github.com/hashicorp:go-multierror",
        "//third_party/go/github.com/hashicorp:go-sockaddr",
        "//third_party/go/github.com/hashicorp:hcl",
        "//third_party/go/github.com/mitchellh:mapstructure",
        "//third_party/go/github.com/pierrec:lz4",
        "//third_party/go/github.com/ryanuber:go-glob",
    ],
)
