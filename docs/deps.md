## Dependency Management

Vendoring all dependencies is essential to have a **go get**-able package.

Tools needed:

- [dep](https://golang.github.io/dep/) dependency manager

`make deps` will install them, in case they are missing

## Add a new dependency

If you write new features which imports a new library, you have to vendor it:
```
dep ensure -v -add github.com/Masterminds/cookoo/web
```

## Add a forked dependency

Sometimes you have an unmerged PR, or a change which you don't even want to push upstream.
In those cases you have a GH fork, and want use that instead of the origin.

Gopkg.toml:
```
[[constraint]]
  name = "github.com/kubicorn/kubicorn"
  branch = "master"
  source = "github.com/banzaicloud/kubicorn"
```

## Update existing dependency

If you are using a specific branch/tag like v1.2.0 in Gopkg.toml, just change it to the 
new version.

Perform an update:

```
dep ensure -v -update github.com/your/upgradable/package

```

## History

This project was previously using [dep](https://github.com/golang/dep). But `dep ensure`
couldn't handle k8s.io dependencies.

This project was previously using [glide](https://github.com/Masterminds/glide). But we returned to dep because seems like
glide is becoming dormant, and seems like dep now can handle k8s.io dependencies.

## Related issues

see GH issues:

- [k8s.io/client-go#83](https://github.com/kubernetes/client-go/issues/83)
- [golang/deps#1207](https://github.com/golang/dep/issues/1207)
