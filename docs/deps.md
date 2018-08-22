# Dependency Management

Pipeline uses [dep](https://golang.github.io/dep/) to vendor dependencies.

Install the latest version from the link above or by running:

```bash
$ make bin/dep # Installs dep to ./bin/dep
```

On MacOS you can install it using Homebrew:

```bash
$ brew install dep
```


## Add a new dependency

If you write new features which imports a new library, you have to vendor it:
```bash
$ dep ensure -v -add github.com/Masterminds/cookoo/web
```


## Add a forked dependency

Sometimes you have an unmerged PR, or a change which you don't even want to push upstream.
In those cases you have a GH fork, and want use that instead of the origin.

Gopkg.toml:
```toml
[[constraint]]
  name = "github.com/kubicorn/kubicorn"
  branch = "master"
  source = "github.com/banzaicloud/kubicorn"
```


## Update existing dependency

If you are using a specific branch/tag like v1.2.0 in Gopkg.toml, just change it to the 
new version.

Perform an update:

```bash
$ dep ensure -v -update github.com/your/upgradable/package
```


## Related issues

see GH issues:

- [k8s.io/client-go#83](https://github.com/kubernetes/client-go/issues/83)
- [golang/deps#1207](https://github.com/golang/dep/issues/1207)
