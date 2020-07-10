package(default_visibility = ["PUBLIC"])

go_get(
    name = "emperror",
    get = "emperror.dev/emperror",
    install = [""],
    sum = "h1:DWdD6gwPaY1FYVmtfBhX/cP+u8g4/W0FdE6w32IerNc=",
    version = "v0.32.0",
    deps = [":errors"],
)

go_get(
    name = "errors",
    get = "emperror.dev/errors",
    install = [
        "",
        "match",
        "utils/keyval",
    ],
    sum = "h1:vf0gZ0j9BJCXQYm0M21DPKeXy9PUjvX1YHXhNzIsqOY=",
    version = "v0.7.0",
    deps = [
        "//third_party/go/github.com/pkg:errors",
        "//third_party/go/go.uber.org:multierr",
    ],
)
