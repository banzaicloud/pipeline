package(default_visibility = ["PUBLIC"])

go_get(
    name = "purell",
    get = "github.com/PuerkitoBio/purell",
    install = [""],
    sum = "h1:WEQqlqaGbrPkxLJWfBwQmfEAE1Z7ONdDLqrN38tNFfI=",
    version = "v1.1.1",
    deps = [
        ":urlesc",
        "//third_party/go/golang.org/x:net",
        "//third_party/go/golang.org/x:text",
    ],
)

go_get(
    name = "urlesc",
    get = "github.com/PuerkitoBio/urlesc",
    install = [""],
    sum = "h1:d+Bc7a5rLufV/sSk/8dngufqelfh6jnri85riMAaF/M=",
    version = "v0.0.0-20170810143723-de5bf2ad4578",
    deps = [],
)
