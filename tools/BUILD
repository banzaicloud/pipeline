subinclude("///pleasings2//go:tools")

# Copied here from https://github.com/sagikazarmark/mypleasings/blob/27b6451ea99d160aec03f242be5261978770b4e1/tools/go/BUILD
# Custom go toolchain doesn't work with subrepos: https://github.com/thought-machine/please/issues/1547
wollemi_wrapper(
    name = "wollemi-wrapper",
    binary = "///pleasings2//tools/go:wollemi",
    labels = ["go"],
    visibility = ["PUBLIC"],
)

remote_file(
    name = "godeps",
    binary = True,
    exported_files = ["godeps"],
    extract = True,
    url = f"https://github.com/sagikazarmark/please-go-modules/releases/download/v0.0.33/godeps_{CONFIG.HOSTOS}_{CONFIG.HOSTARCH}.tar.gz",
)

sh_cmd(
    name = "plz-tidy",
    cmd = [
        "$(out_exe //tools:godeps) -dir third_party/go -clean -builtin -wollemi",
        "$(out_exe :wollemi-wrapper) gofmt",
        "$(out_exe :wollemi-wrapper) gofmt ./internal/.generated/...",
    ],
    deps = [
        ":wollemi-wrapper",
        "//tools:godeps",
    ],
)
