## Dependency Management

Vendoring all dependencies is essential to have a **go get**-able package.

Tools needed:

- [glide](github.com/Masterminds/glide) dependency manager
- [glide-vc](github.com/sgotti/glide-vc) vendor cleaner plugin for glide

`make deps` will instal them, in case they are missing

## Add a new dependency

If you write new features which imports a new library, you have to vendor it:
```
glide get -v github.com/Masterminds/cookoo/web
glide-vc --only-code --no-tests
```

## Add a forked dependency

Sometimes you have an unmerged PR, or a change which you don't even want to push upstream.
In those cases you have a GH fork, and want use that instead of the origin.

glide.yaml:
```
- package: k8s.io/helm
  repo: https://github.com/banzaicloud/helm.git
  vcs: git
```

## Update existing dependency

If you are using a specific branch/tag like v1.2.0 in glide.yaml, just change it to the 
new version.

Otherwise if you haven't picked a branch/tag and just used the latest master, you will
have a commit sha in glide.lock. Due to a [bug in glide](https://github.com/Masterminds/glide/issues/592)
it will be stuck on that sha.
To fix it, you have to specify **master** (or a branch/tag) in **glide.yaml**:
```
- package: github.com/prometheus/prometheus
  version: ^2.0.0
```

than perform an update:

```
glide up -v
glide-vc --only-code --no-tests
```

## History

This project was previously using [dep](https://github.com/golang/dep). But `dep ensure`
couldn't handle k8s.io dependencies.

## Related issues

see GH issues:

- [k8s.io/client-go#83](https://github.com/kubernetes/client-go/issues/83)
- [golang/deps#1207](https://github.com/golang/dep/issues/1207)
