# Pipeline

[![CircleCI](https://circleci.com/gh/banzaicloud/pipeline/tree/master.svg?style=shield)](https://circleci.com/gh/banzaicloud/pipeline/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/banzaicloud/pipeline)](https://goreportcard.com/report/github.com/banzaicloud/pipeline)
[![CII Best Practices](https://bestpractices.coreinfrastructure.org/projects/1651/badge)](https://bestpractices.coreinfrastructure.org/projects/1651)
[![Docker Automated build](https://img.shields.io/docker/automated/banzaicloud/pipeline.svg)](https://hub.docker.com/r/banzaicloud/pipeline/)

**Pipeline enables developers to go from commit to scale in minutes by turning Kubernetes into a feature rich application platform integrating CI/CD, centralized logging, monitoring, enterprise-grade security, cost management and autoscaling.**

> _Banzai Pipeline, or simply Pipeline is a tabletop reef break located in Hawaii, Oahu's North Shore. The most famous and infamous reef in the universe is the benchmark by which all other waves are measured._


## What is Pipeline?

Pipeline is a feature rich **application platform**, built for containers on top of Kubernetes to automate the DevOps experience, continuous application development and the lifecycle of deployments. Pipeline enables developers to go from commit to scale in minutes by turning Kubernetes into a feature rich application platform integrating CI/CD, centralized logging, monitoring, enterprise-grade security and autoscaling.

The main features of the platform are:

* **Provisioning:** _Provision highly available Kubernetes clusters on any of the supported cloud providers, on-premise or hybrid configurations_
* **Application focused:** _Focus on building great applications and leave the hard stuff of ops, failover, build pipelines, patching and security to Banzai Pipeline_
* **Scaling:** _Supports SLA rules for resiliency, failover and autoscaling_
* **Observability:** _Centralized log collection, tracing and advanced monitoring support for the infrastructure, Kubernetes cluster and the deployed applications_
* **Hook in:** _Go from commit to scale in minutes using our container-native CI/CD workflow engine_
* **Spotguides:** _Use your favorite development framework and let Pipeline automate the rest_

Check out the developer beta if you would like to try out the platform:
<p align="center">
  <a href="https://try.banzaicloud.io">
  <img src="https://camo.githubusercontent.com/a487fb3128bcd1ef9fc1bf97ead8d6d6a442049a/68747470733a2f2f62616e7a6169636c6f75642e636f6d2f696d672f7472795f706970656c696e655f627574746f6e2e737667">
  </a>
</p>


## Installation

Pipeline can be installed locally for development by following the [development guide](docs/pipeline-howto.md).

If you want to install Pipeline for production usage, please read our [installation guide](https://banzaicloud.com/docs/pipeline/quickstart/install-pipeline/).


## Architecture overview

Pipeline enforces a typical **cloud native** architecture which takes full advantage of on-demand delivery, global deployment, elasticity, and higher-level services. It enables huge improvements in developer productivity, business agility, scalability, availability, utilization, and cost savings.

It is written in Golang and built on public cloud provider APIs, Kubernetes, Helm, Prometheus, Grafana, Docker, Vault and a few other open source technologies from the CNCF landscape - however all of these are abstracted for the end user behind a secure REST API, UI or CLI.

### Control plane

The Pipeline Control Plane is the central location where all the components of the [Pipeline Platform](https://banzaicloud.com/platform/) are assembled together and it runs all the provided services as CI/CD, authentication, log collection, monitoring, dashboards, application registries, spotguide definitions, security scans and more. The control plane itself is a Kubernetes deployment as well, and it's cloud agnostic - currently there are out of the box deployments for [AWS](https://github.com/banzaicloud/pipeline-cp-launcher/blob/master/README.md#pipeline-control-plane-launcher-on-aws), [Azure](https://github.com/banzaicloud/pipeline-cp-launcher/blob/master/README.md#pipeline-control-plane-launcher-on-azure), [Google](https://github.com/banzaicloud/pipeline-cp-launcher/blob/master/README.md#pipeline-control-plane-launcher-on-google-cloud) and for [Minikube](https://github.com/banzaicloud/pipeline-cp-launcher/blob/master/README.md#pipeline-control-plane-launcher-on-minikube) (for local/dev purpose).

A default control plane deployment looks like this:

<p align="center">
<img src="docs/images/multi.png" width="700">
</p>


### Contributing

Thanks you for your contribution and being part of our community.

Please read [CONTRIBUTING.md](https://github.com/banzaicloud/.github/blob/master/CONTRIBUTING.md) for details on the code of conduct,
and the process for submitting pull requests. When you are opening a PR to Pipeline the first time we will require you to sign a standard CLA.


### License

Copyright (c) 2017-2019 [Banzai Cloud, Inc.](https://banzaicloud.com)

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

[http://www.apache.org/licenses/LICENSE-2.0](http://www.apache.org/licenses/LICENSE-2.0)

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
