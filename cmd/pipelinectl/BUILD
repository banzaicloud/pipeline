subinclude("///pleasings2//go")

#go_build(
#    name = "pipelinectl2",
#    definitions = {
#        "main.version": "${VERSION:-" + git_branch() + "}",
#        "main.commitHash": git_commit()[0:8],
#        "main.buildDate": "$SCM_COMMIT_DATE",
#    },
#    pass_env = ["VERSION"],
#    stamp = True,
#    trimpath = True,
#    label = ["manual"],
#)

go_binary(
    name = "pipelinectl",
    srcs = glob(["*.go"], exclude = ["*_test.go"]),
    visibility = ["PUBLIC"],
    deps = [
        "//internal/app/pipelinectl/cli/commands",
        "//third_party/go/github.com/spf13/cobra",
        "//third_party/go/github.com/spf13/viper",
    ],
)
