filegroup(
    name = "config.dev.yaml",
    srcs = ["config.dev.yaml"],
    visibility = ["PUBLIC"],
)

genrule(
    name = "config.yaml",
    srcs = ["config.dev.yaml"],
    outs = ["config/config.yaml"],
    cmd = 'cat config/config.dev.yaml | sed "s/uuid: \\"\\"/uuid: \\"$RANDOM.$USER.local\\"/" > "$OUTS"',
    pass_env = ["USER"],
)
