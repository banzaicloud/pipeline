# Helm 3

This document outlines the steps necessary to upgrade Pipeline to [Helm 3](https://helm.sh/blog/helm-3-released/).


## Repository management

Currently repository information is stored on the filesystem (where Helm would normally store it).
Unfortunately, Pipeline does not persist this information between deployments.

The solution is rewriting the current repository management API to store repositories in the database.
In addition, we also need to be able to attach optional username-password and TLS certificates to repositories using the Secret API
(Password and TLS type secrets).

### API

Calls:

```
GET    /helm/repos           - List helm repos
POST   /helm/repos           - Add new repo
PUT    /helm/repos/:repoName - Update repo
DELETE /helm/repos/:repoName - Delete repo
```

Repository model:

```json
{
    "name": "my-repo",
    "url": "https://charts.example.com",
    "passwordSecretId": "dc460da4ad72c482231e28e688e01f2778a88ce31a08826899d54ef7183998b5",
    "tlsSecretId": "dc460da4ad72c482231e28e688e01f2778a88ce31a08826899d54ef7183998b5"
}
```

### Decisions to be made

#### Builtin repositories

Certain repositories are used internally by Pipeline to install its own components and integrated services on clusters.
Currently, users can edit these repositories, but we need to ensure that every Pipeline deployment comes with the latest version of these repositories
(whereas users can manually update repositories). This can potentially lead to failed Helm installs when a repository is not up to date.

By introducing "builtin repositories" we can make sure that Pipeline internally uses different, up to date repositories for installing its own components.


## Index cache

Repository index cache is currently stored on the filesystem (where Helm would normally store it).
By definition, the local repository index file is a cache (similarly to other package managers),
so saving it in persistent storage is not necessary.
On the other hand, for performance reasons it would make sense to store it somewhere that's not lost on every deployment.
(The state store where Helm currently stores its state is not persisted between deployments.)

### API

Calls:

```
GET   /helm/repos/:repoName/index.yaml - Get repo index
PURGE /helm/repos/:repoName/index.yaml - Purge the repo index cache (update it)
```

### Decisions to be made

#### Return the repository index in the API

Returning the index file allows tools, like `banzai` CLI to use the same repository index as Pipeline for installing charts.

#### Persist index cache in the database

The index cache is not updated automatically at the moment, users have to update it by sending a request to the API.
That might suggest the index file is not actually a cache, but a "versioned" resource that shouldn't be volatile,
in which case it needs to be persisted.


## Chart cache

Currently we use Helm library tools to manage the cache the way the Helm CLI normally does (in `$HELM_HOME`, following a certain directory structure).
After the repository and index cache management is (re)written, we need to rewrite the chart cache management,
so we can drop the Helm home concept and replace it with a simplified, service oriented concept.


## Chart detail cache

Every time we load charts, we need to unpack them and parse the `Chart.yaml` file. This is slow, because of the lot of IO operations.
We should implement a cache layer for chart details, so that they can be loaded fast, without parsing each chart every time.

### Decisions to be made

#### Clear the chart detail cache every time the index cache is purged

Should we do that?


## Deployment API

TODO


## Upgrade Helm to version 3

TODO
