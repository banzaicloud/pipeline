# Pipeline

Banzai Pipeline, or simply [Pipeline](https://banzaicloud.io) is a tabletop reef break located in Hawaii, Oahu's North Shore. The most famous and infamous reef in the universe is the benchmark by which all other waves are measured.

Pipeline enables developers to go from commit to scale in minutes by turning Kubernetes into a feature rich application platform integrating CI/CD, centralized logging, monitoring, enterprise-grade security, cost management and autoscaling.

## TL;DR;

```bash
$ helm repo add banzaicloud-stable https://kubernetes-charts.banzaicloud.com
$ helm repo update
```

## Introduction

This chart bootstraps a [Pipeline](https://github.com/banzaicloud/pipeline) deployment on a [Kubernetes](http://kubernetes.io) cluster using the [Helm](https://helm.sh) package manager.

## Prerequisites

- Kubernetes 1.10+ with Beta APIs enabled

## Installing the Chart

To install the chart with the release name `my-release`:

```bash
$ helm install --name my-release --namespace banzaicloud banzaicloud-stable/pipeline
```

## Uninstalling the Chart

To uninstall/delete the `my-release` deployment:

```bash
$ helm delete my-release
```

The command removes all the Kubernetes components associated with the chart and deletes the release.

## Configuration (Database)

The following table lists the configurable parameters of the Pipeline chart database configuration and their default values.

#### Postgres (default)

Read more [stable/postgresql](https://github.com/helm/charts/tree/master/stable/postgresql)

```yaml
postgres:
  enabled: true
```

| Parameter        | Description              | Default  |
| ---------------- | ------------------------ | -------- |
| postgres.enabled | Install postgresql chart | true     |

#### Mysql

Read more [stable/mysql](https://github.com/helm/charts/tree/master/stable/mysql)

```yaml
mysql:
  enabled: true
```

| Parameter     | Description         | Default  |
| ------------- | ------------------- | -------- |
| mysql.enabled | Install mysql chart | false    |

#### Custom settings (These `values` ​​are preferred against mysql or postgres `values`)

| Parameter               | Description                                   | Default       |
| ------------------------| --------------------------------------------- | ------------- |
| database.driver         | Database driver (mysql, postgres)             | ``            |
| database.host           | Database host                                 | ``            |
| database.port           | Database port                                 | ``            |
| database.tls            | Database TLS parameter                        | `turned off`  |
| database.name           | Database name                                 | `pipeline`    |
| database.username       | Database username                             | `pipeline-rw` |
| database.password       | Database password                             | ``            |
| database.existingSecret | Use an existing secret for database passwords | ``            |

#### Setting up Google CloudSQL Proxy

Read more [rimusz/gcloud-sqlproxy](https://github.com/rimusz/charts/tree/master/stable/gcloud-sqlproxy)

```yaml
cloudsql:
  enabled: true
    instances: []
#      - project:
#        region: 
#        instance:
#        port:
```

| Parameter        | Description            | Default  |
| ---------------- | ---------------------- | -------- |
| cloudsql.enabled | Install cloudsql chart | false    |


Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example:

```console
$ helm install --name my-release --set server.image.tag=0.17.0 banzaicloud-stable/pipeline
```

Alternatively, a YAML file that specifies the values for the parameters can be provided while
installing the chart. For example:

```console
$ helm install --name my-release --values values.yaml banzaicloud-stable/pipeline
```

