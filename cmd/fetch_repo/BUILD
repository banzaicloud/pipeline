go_binary(
    name = "fetch_repo",
    srcs = [
        "fetch_repo.go",
        "module.go",
        "vcs.go",
    ],
    visibility = ["PUBLIC"],
    deps = ["//third_party/go:golang.org_x_tools"],
)

go_test(
    name = "test",
    srcs = [
        "fetch_repo.go",
        "fetch_repo_test.go",
        "module.go",
        "vcs.go",
    ],
    deps = ["//third_party/go:golang.org_x_tools"],
)

go_get(
    name = "tools",
    get = "golang.org/x/tools",
    install = ["go/vcs"],
    revision = "aa94e735be7f",
)
