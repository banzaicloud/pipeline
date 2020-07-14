# moddown: Go Module downloader

`moddown` is a simplified version of [fetch_repo](https://github.com/bazelbuild/bazel-gazelle/tree/5c00b77/cmd/fetch_repo). It focuses on downloading a module using `go mod download`.
Unlike `fetch_repo`, this tool [does not create a dummy module](https://github.com/golang/go/issues/29522)
and uses the [-modcacherw](https://golang.org/doc/go1.14#go-flags) flag to make the cache writable (removable),
so it requires at least Go 1.14.
