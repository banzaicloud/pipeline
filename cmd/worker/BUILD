subinclude("///pleasings2//go")

go_build(
    name = "worker",
    definitions = {
        "main.version": "${VERSION:-" + git_branch() + "}",
        "main.commitHash": git_commit()[0:8],
        "main.buildDate": "$SCM_COMMIT_DATE",
    },
    pass_env = ["VERSION"],
    stamp = True,
    trimpath = True,
)
