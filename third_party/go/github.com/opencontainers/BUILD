package(default_visibility = ["PUBLIC"])

go_get(
    name = "go-digest",
    get = "github.com/opencontainers/go-digest",
    install = [""],
    sum = "h1:WzifXhOVOEOuFYOJAW6aQqW0TooG2iki3E3Ii+WN7gQ=",
    version = "v1.0.0-rc1",
    deps = [],
)

go_get(
    name = "image-spec",
    get = "github.com/opencontainers/image-spec",
    install = [
        "specs-go",
        "specs-go/v1",
    ],
    sum = "h1:JMemWkRwHx4Zj+fVxWoMCFm/8sYGGrUVojFA6h/TRcI=",
    version = "v1.0.1",
    deps = [":go-digest"],
)

go_get(
    name = "runc",
    get = "github.com/opencontainers/runc",
    install = [
        "libcontainer/user",
        "libcontainer/system",
    ],
    sum = "h1:/k06BMULKF5hidyoZymkoDCzdJzltZpz/UU4LguQVtc=",
    version = "v1.0.0-rc9",
    deps = ["//third_party/go/golang.org/x:sys"],
)
