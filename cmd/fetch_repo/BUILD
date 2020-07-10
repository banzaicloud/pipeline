go_binary(
    name = "fetch_repo",
    srcs = [
        "fetch_repo.go",
        "module.go",
        "vcs.go",
    ],
    visibility = ["PUBLIC"],
    deps = [":tools"],
)

go_test(
    name = "test",
    srcs = [
        "fetch_repo.go",
        "fetch_repo_test.go",
        "module.go",
        "vcs.go",
    ],
    deps = [":tools"],
)

go_get(
    name = "tools",
    get = "golang.org/x/tools",
    install = ["go/vcs"],
    revision = "aa94e735be7f",
)
