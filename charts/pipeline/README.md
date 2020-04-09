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

- Kubernetes 1.12+

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

| Parameter               | Description                                   | Default   |
| ------------------------| --------------------------------------------- | --------- |
| database.driver         | Database driver (mysql, postgres)             | ``        |
| database.host           | Database host                                 | ``        |
| database.port           | Database port                                 | ``        |
| database.tls            | Database TLS parameter                        | ``        |
| database.name           | Database name                                 | `pipeline`|
| database.username       | Database username                             | `pipeline`|
| database.password       | Database password                             | ``        |
| database.existingSecret | Use an existing secret for database passwords | ``        |

Specify each parameter using the `--set key=value[,key=value]` argument to `helm install`. For example:

```console
$ helm install --name my-release --set server.image.tag=0.40.0 banzaicloud-stable/pipeline
```

Alternatively, a YAML file that specifies the values for the parameters can be provided while
installing the chart. For example:

```console
$ helm install --name my-release --values values.yaml banzaicloud-stable/pipeline
```

