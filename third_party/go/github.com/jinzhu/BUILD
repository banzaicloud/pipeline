package(default_visibility = ["PUBLIC"])

go_get(
    name = "copier",
    get = "github.com/jinzhu/copier",
    install = [""],
    sum = "h1:zPPuIq2jAWWPTrGt70eK/BSch+gFAGrNzecsoENgu2o=",
    version = "v0.0.0-20190924061706-b57f9002281a",
    deps = [],
)

go_get(
    name = "gorm",
    get = "github.com/jinzhu/gorm",
    install = [
        "",
        "dialects/mysql",
        "dialects/postgres",
        "dialects/sqlite",
    ],
    replace = "github.com/jinzhu/gorm",
    sum = "h1:lDSDtsCt5AGGSKTs8AHlSDbbgif4G4+CKJ8ETBDVHTA=",
    version = "v1.9.1",
    deps = [
        ":inflection",
        "//third_party/go/github.com/go-sql-driver:mysql",
        "//third_party/go/github.com/lib:pq",
        "//third_party/go/github.com/mattn:go-sqlite3",
    ],
)

go_get(
    name = "inflection",
    get = "github.com/jinzhu/inflection",
    install = [""],
    sum = "h1:K317FqzuhWc8YvSVlFMCCUb36O/S9MCKRDI7QkRKD/E=",
    version = "v1.0.0",
    deps = [],
)

go_get(
    name = "now",
    get = "github.com/jinzhu/now",
    install = [""],
    sum = "h1:g39TucaRWyV3dwDO++eEc6qf8TVIQ/Da48WmqjZ3i7E=",
    version = "v1.1.1",
    deps = [],
)
