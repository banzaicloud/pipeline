subinclude("///pleasings2//go")

go_build(
    name = "pipeline",
    definitions = {
        "main.version": "${VERSION:-" + git_branch() + "}",
        "main.commitHash": git_commit()[0:8],
        "main.buildDate": "$SCM_COMMIT_DATE",
        "main.helmVersion": '$(cat go.mod | grep helm.sh/helm/v3 | grep -v "=>" | cut -d" " -f2 | sed s/^v//)',
    },
    pass_env = ["VERSION"],
    stamp = True,
    trimpath = True,
)
