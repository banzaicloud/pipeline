package(default_visibility = ["PUBLIC"])

go_get(
    name = "logur",
    get = "emperror.dev/handler/logur",
    install = [""],
    sum = "h1:mo12Nr3Ke3llow2lNi9PuS0h4vy24yDoAQ8bw98v1pY=",
    version = "v0.4.0",
    deps = [
        "//third_party/go/emperror.dev:errors",
        "//third_party/go/logur.dev:logur",
    ],
)

go_get(
    name = "stackdriver",
    get = "emperror.dev/handler/stackdriver",
    install = [""],
    sum = "h1:wUoZUU7ee/UsDWLnwLpkOPrNp7lQpZ1pQfN507QNGfs=",
    version = "v0.3.0",
    deps = [
        "//third_party/go/cloud.google.com/go:errorreporting",
        "//third_party/go/emperror.dev:errors",
    ],
)
