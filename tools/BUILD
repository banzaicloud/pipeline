subinclude("///pleasings2//go:tools")

# Copied here from https://github.com/sagikazarmark/mypleasings/blob/27b6451ea99d160aec03f242be5261978770b4e1/tools/go/BUILD
# Custom go toolchain doesn't work with subrepos: https://github.com/thought-machine/please/issues/1547
wollemi_wrapper(
    name = "wollemi-wrapper",
    binary = "///pleasings2//tools/go:wollemi",
    labels = ["go"],
    visibility = ["PUBLIC"],
)

sh_cmd(
    name = "plz-tidy2",
    cmd = [
        "$(out_exe ///pleasegomod//:godeps) -dir third_party/go -clean -builtin -wollemi",
        "$(out_exe :wollemi-wrapper) gofmt",
        "$(out_exe :wollemi-wrapper) gofmt ./internal/.generated/...",
    ],
    deps = [
        ":wollemi-wrapper",
        "///pleasegomod//:godeps",
    ],
)

remote_file(
    name = "godeps",
    binary = True,
    exported_files = ["godeps"],
    extract = True,
    hashes = [
        "21dfcb8cacab410c351e3e57f3dbb79d77621da4bde42fad180f9d6ad82123c3",
        "16acdca286a85e73f2643ec2726f7fa749f0c657b92b4e67eeec43c275854b63",
    ],
    url = f"https://github.com/sagikazarmark/please-go-modules/releases/download/v0.0.31/godeps_{CONFIG.HOSTOS}_{CONFIG.HOSTARCH}.tar.gz",
)

remote_file(
    name = "wollemi",
    binary = True,
    exported_files = ["wollemi"],
    extract = True,
    url = f"https://github.com/tcncloud/wollemi/releases/download/v{CONFIG.WOLLEMI_VERSION}/wollemi-v{CONFIG.WOLLEMI_VERSION}-{CONFIG.HOSTOS}-{CONFIG.HOSTARCH}.tar.gz",
)

sh_cmd(
    name = "plz-tidy",
    cmd = [
        "$(out_exe :godeps) -dir third_party/go -clean -wollemi -noexpand",
        "$(out_exe :wollemi) gofmt",
        #"$(out_exe :wollemi) gofmt ./internal/.generated/...",
    ],
    deps = [
        ":godeps",
        ":wollemi",
    ],
)
