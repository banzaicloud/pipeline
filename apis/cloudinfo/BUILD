subinclude("///pleasings2//openapi")

cloudinfo_version = "0.9.5"

genrule(
    name = "spec",
    srcs = [remote_file(
        name = "spec",
        _tag = "download",
        url = f"https://raw.githubusercontent.com/banzaicloud/cloudinfo/{cloudinfo_version}/api/openapi-spec/cloudinfo.yaml",
        hashes = ["f564d5398c1414712e9ef65a69d48aa8986470a6b3e35f806e8007d076b12735"],
        licences = ["Apache-2.0"],
    )],
    outs = ["apis/cloudinfo/openapi.yaml"],
    cmd = f'cat "$SRCS" | sed "s/version: .*/version: {cloudinfo_version}/" > "$OUTS"',
)

openapi_library(
    name = "cloudinfo",
    spec = ":spec",
    generator = "go",
    out = ".gen/cloudinfo",
    additional_properties = {
        "packageName": "cloudinfo",
        "withGoCodegenComment": "true",
        "sourceFolder": "cloudinfo",
    },
    strip = ["{.travis.yml,git_push.sh,go.*,docs}"],
)
