package(default_visibility = ["PUBLIC"])

go_get(
    name = "errwrap",
    get = "github.com/hashicorp/errwrap",
    install = [""],
    sum = "h1:hLrqtEDnRye3+sgx6z4qVLNuviH3MR5aQ0ykNJa/UYA=",
    version = "v1.0.0",
    deps = [],
)

go_get(
    name = "go-cleanhttp",
    get = "github.com/hashicorp/go-cleanhttp",
    install = [""],
    sum = "h1:dH3aiDG9Jvb5r5+bYHsikaOUIpcM0xvgMXVoDkXMzJM=",
    version = "v0.5.1",
    deps = [],
)

go_get(
    name = "go-multierror",
    get = "github.com/hashicorp/go-multierror",
    install = [""],
    sum = "h1:iVjPR7a6H0tWELX5NxNe7bYopibicUzc7uPribsnS6o=",
    version = "v1.0.0",
    deps = [":errwrap"],
)

go_get(
    name = "go-retryablehttp",
    get = "github.com/hashicorp/go-retryablehttp",
    install = [""],
    sum = "h1:1BZvpawXoJCWX6pNtow9+rpEj+3itIlutiqnntI6jOE=",
    version = "v0.5.4",
    deps = [":go-cleanhttp"],
)

go_get(
    name = "go-rootcerts",
    get = "github.com/hashicorp/go-rootcerts",
    install = [""],
    sum = "h1:DMo4fmknnz0E0evoNYnV48RjWndOsmd6OW+09R3cEP8=",
    version = "v1.0.1",
    deps = ["//third_party/go/github.com/mitchellh:go-homedir"],
)

go_get(
    name = "go-sockaddr",
    get = "github.com/hashicorp/go-sockaddr",
    install = [""],
    sum = "h1:ztczhD1jLxIRjVejw8gFomI1BQZOe2WoVOu0SyteCQc=",
    version = "v1.0.2",
    deps = [],
)

go_get(
    name = "golang-lru",
    get = "github.com/hashicorp/golang-lru",
    install = [
        "simplelru",
        "",
    ],
    sum = "h1:0hERBMJE1eitiLkihrMvRVBYAkpHzc/J3QdDN+dAcgU=",
    version = "v0.5.1",
    deps = [],
)

go_get(
    name = "hcl",
    get = "github.com/hashicorp/hcl",
    install = [
        "hcl/strconv",
        "hcl/token",
        "hcl/ast",
        "hcl/scanner",
        "hcl/parser",
        "json/token",
        "json/scanner",
        "json/parser",
        "",
        "hcl/printer",
    ],
    sum = "h1:0Anlzjpi4vEasTeNFn2mLJgTSwt0+6sfsiTG8qcWGx4=",
    version = "v1.0.0",
    deps = [],
)
