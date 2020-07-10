package(default_visibility = ["PUBLIC"])

go_get(
    name = "logrus",
    get = "logur.dev/adapter/logrus",
    install = [""],
    sum = "h1:qBw5g87m++zoFOq6K30J+UnQC7bZrPAwpmoF3G7Oxo8=",
    version = "v0.4.1",
    deps = [
        "//third_party/go/github.com/sirupsen:logrus",
        "//third_party/go/logur.dev:logur",
    ],
)

go_get(
    name = "zap",
    get = "logur.dev/adapter/zap",
    install = [
        "internal/keyvals",
        "",
    ],
    sum = "h1:lZFwPNU3SQamoxejhkU2v/TTt/vC/eeYQBO9MIPKXmc=",
    version = "v0.4.1",
    deps = [
        "//third_party/go/go.uber.org:zap",
        "//third_party/go/logur.dev:logur",
    ],
)
