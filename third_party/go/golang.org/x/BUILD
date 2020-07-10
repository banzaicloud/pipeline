package(default_visibility = ["PUBLIC"])

go_get(
    name = "crypto",
    get = "golang.org/x/crypto",
    install = [
        "ed25519",
        "pbkdf2",
        "ssh/terminal",
        "pkcs12/internal/rc2",
        "pkcs12",
        "internal/subtle",
        "chacha20",
        "curve25519",
        "poly1305",
        "blowfish",
        "ssh/internal/bcrypt_pbkdf",
        "ssh",
        "scrypt",
        "openpgp/errors",
        "openpgp/armor",
        "cast5",
        "openpgp/elgamal",
        "openpgp/s2k",
        "openpgp/packet",
        "openpgp",
        "openpgp/clearsign",
        "bcrypt",
    ],
    sum = "h1:/Tl7pH94bvbAAHBdZJT947M/+gp0+CqQXDtMRC0fseo=",
    version = "v0.0.0-20200220183623-bac4c82f6975",
    deps = [":sys"],
)

go_get(
    name = "net",
    get = "golang.org/x/net",
    install = [
        "idna",
        "http/httpguts",
        "http2/hpack",
        "http2",
        "context/ctxhttp",
        "internal/timeseries",
        "trace",
        "context",
        "bpf",
        "internal/iana",
        "internal/socket",
        "ipv4",
        "ipv6",
        "html/atom",
        "html",
        "publicsuffix",
    ],
    sum = "h1:QPwSajcTUrFriMF1nJ3XzgoqakqQEsnZf9LdXdi2nkI=",
    version = "v0.0.0-20200421231249-e086a090c8fd",
    deps = [
        ":sys",
        ":text",
    ],
)

go_get(
    name = "oauth2",
    get = "golang.org/x/oauth2",
    install = [
        "internal",
        "",
        "jws",
        "jwt",
        "google",
    ],
    sum = "h1:TzXSXBo42m9gQenoE3b9BGiEpg5IG2JkU5FkPIawgtw=",
    version = "v0.0.0-20200107190931-bf48bf16ab8d",
    deps = [
        ":net",
        "//third_party/go/cloud.google.com/go/compute:metadata",
    ],
)

go_get(
    name = "sync",
    get = "golang.org/x/sync",
    install = [
        "errgroup",
        "semaphore",
    ],
    sum = "h1:vcxGaoTs7kV8m5Np9uUNQin4BrLOthgV7252N8V+FwY=",
    version = "v0.0.0-20190911185100-cd5d95a43a6e",
    deps = [],
)

go_get(
    name = "sys",
    get = "golang.org/x/sys",
    install = ["unix"],
    sum = "h1:xhmwyvizuTgC2qz7ZlMluP20uW+C3Rm0FD/WLDX8884=",
    version = "v0.0.0-20200323222414-85ca7c5b95cd",
    deps = [],
)

go_get(
    name = "text",
    get = "golang.org/x/text",
    install = [
        "transform",
        "unicode/bidi",
        "secure/bidirule",
        "unicode/norm",
        "encoding/internal/identifier",
        "encoding",
        "encoding/internal",
        "internal/utf8internal",
        "runes",
        "encoding/unicode",
        "width",
    ],
    sum = "h1:tW2bmiBqwgJj/UpqtC8EpXEZVYOwU0yG4iWbprSVAcs=",
    version = "v0.3.2",
    deps = [],
)

go_get(
    name = "time",
    get = "golang.org/x/time",
    install = ["rate"],
    sum = "h1:NusfzzA6yGQ+ua51ck7E3omNUX/JuqbFSaRGqU8CcLI=",
    version = "v0.0.0-20200416051211-89c76fbcd5d1",
    deps = [],
)

go_get(
    name = "tools",
    get = "golang.org/x/tools",
    install = ["go/vcs"],
    sum = "h1:0sfSpGSa544Fwnbot3Oxq/U6SXqjty6Jy/3wRhVS7ig=",
    version = "v0.0.0-20200216192241-b320d3a0f5a2",
    deps = [],
)
