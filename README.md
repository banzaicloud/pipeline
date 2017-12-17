[![CircleCI](https://circleci.com/gh/banzaicloud/pipeline/tree/master.svg?style=svg)](https://circleci.com/gh/banzaicloud/pipeline/tree/master)
[![Go Report Card](https://goreportcard.com/badge/github.com/banzaicloud/pipeline)](https://goreportcard.com/report/github.com/banzaicloud/pipeline)


_Banzai Pipeline, or simply Pipeline is a tabletop reef break located in Hawaii, Oahu's North Shore. The most famous and infamous reef on the planet is forming the benchmark by which all other waves are measured._

_Pipeline is a RESTful API to deploy **cloud native** microservices in public cloud and on-premise. It simplifies and abstracts all the details of provisioning the cloud infrastructure, installing or reusing the Kubernetes cluster and deploying the application._

<p align="center">
  <img width="139" height="197" src="docs/images/warning.jpg">
</p>

**Warning:** _Pipeline is experimental, under development and does not have a stable release yet. If in doubt, don't go out._

- [Pipeline](#pipeline)
  - [Cloud providers](#cloud-providers)
  - [Managed Kubernetes](#managed-kubernetes)
- [Architecture overview](#architecture-overview)
  - [Control plane](#control-plane)
  - [Deployed cluster](#deployed-cluster)
  - [The Pipeline Platform - PaaS](#the-pipeline-platform-paas)
- [Installation](#installation)
- [Create and scale your cluster](#create-and-scale-your-cluster)
- [Quick howto](#pipeline-howto)
- [Spotguide specification](#spotguide-specification)
  - [Big data](#big-data)
  - [Apache Spark](#apache-spark)
  - [Apache Zeppelin](#apache-zeppelin)
  - [Apache Kafka](#apache-kafka)
- [Reporting bugs](#reporting-bugs)
- [Contributing](#contributing)
- [License](#license)

# Pipeline

The Pipeline API is able to create and configure custom Kubernetes clusters in the cloud and deploy cloud native applications while taking care about services, ingress, network and volume management, replication and failover. It also monitors and autoscales the underlying cloud infrastucture, K8S cluster, pods and the application itself.

It is a core building block and engine of the **Pipeline Platform** - an open source, multi cloud application platform as a service, based on a container management system. It is an application centric approach of a software delivery and deployment pipeline, to build cloud native microservices with confidence. It simplifies and abstracts all the details of provisioning the cloud infrastructure, installing or reusing the managed Kubernetes cluster and building, deploying and monitoring the application. The PaaS is a container management platform including a full CI/CD workflow, orchestration, SLA rules based autoscaling and advanced networking delivering a complete solution for developers.

The main features of the platform are:

* **Provisioning:** _Provision highly available clusters on any of the supported cloud providers or managed Kubernetes commercial offerings, hybrid cloud or on-premise deployments_
* **Microservice Ready:** _Focus and build great applications and forget the hard stuff of ops, failover, build pipelines, patching and security_
* **Scaling:** _Supports SLA rules for resiliency, failover and autoscaling_
* **Operations support:** _Centralized log collection, tracing and advanced monitoring support for the underlying infrastructure, Kubernetes cluster and the deployed application_
* **Spotguides:** _Understands the application runtime requirements and dependencies, builds the artifacts and pushes to the PaaS while applying the CI/CD pipeline steps and advanced features (SLA rules, monitoring, linear regression based predictions)_
* **Hook in:** _Trigger your pipeline with a GitHub Webhook and let the PaaS to build your app, provision or reuse the infrastructure and deploy, monitor and autoscale your deployment_

The platform includes a few default `spotguides` like: **Apache Spark, Apache Zeppelin** and **Apache Kafka.**

## Cloud Providers

Pipeline is currently experimental and all the development, testing and the CI/CD pipeline itself is tested on **AWS** only. The AWS version contains the control plane, cloud images, Cloudformation templates and belonging artifacts published.

The underlying [Kubicorn](http://kubicorn.io) framework has support for the following providers.

  * Amazon AWS
  * Google Cloud
  * Microsoft Azure
  * Digital Ocean

## Managed Kubernetes

Pipeline is architected in a way to allow pluggable implementations for providers, managed Kubernetes clusters or hybrid environments. Through provider plugins retrives the `kubeconfig` and connects and deploys applications. Currently it's tested with [Microsoft's Azure managed Kubernetes](https://azure.microsoft.com/en-us/blog/introducing-azure-container-service-aks-managed-kubernetes-and-azure-container-registry-geo-replication/).

  * Microsoft AKS

 There is work planned for the following plugins:

  * CoreOS Tectonic
  * Redhat OpenShift
  * Google GKE

## Architecture overview

Pipeline enforces a typical **cloud native** architecture which takes full advantage of on-demand delivery, global deployment, elasticity, and higher-level services. It enables huge improvements in developer productivity, business agility, scalability, availability, utilization, and cost savings.

It is written in `Golang` and built on public cloud provider APIs, Kubernetes, Kubicorn, Helm, Prometheus, Drone, Docker and a few other open source technologies - however all of these are abstracted for the end user behind a secure REST API.
The central component of the Pipeline API is [Kubicorn](http://kubicorn.io) - a library/cli tool to provision and reconcile Kubernetes clusters in the cloud. Once the infrastructure/K8S cluster is provisioned by Pipeline, it also orchestrates the deployment of the application using Helm charts.
A cluster can be reused and new and updated charts can be redeployed as well - without the need of recreating the cloud infrastructure or the K8S cluster. It supports alerting and autoscaling based on metrics using Prometheus.
By default there are metrics and Grafana dashboards exposing the behaviour of the underlying infrastructure, Kubernetes cluster and the application itself - based on the application specific `spotguide`.

### Control plane

All these components are assembled into a **Control Plane** - and deployed to Kubernetes with Helm. A typical control plane - for an out of the box Spark/Zeppelin `spotguide` - looks like this:

![Control Plane](docs/images/control-plane-aws.png)

You can launch a Pipeline control plane on AWS with the following [Cloudformation](https://github.com/banzaicloud/pipeline-cp-launcher/blob/0.1.0/control-plane.template) template.

### Deployed cluster

A typical cluster/application deployed with Pipeline - for an out of the box Spark/Zeppelin spotguide - looks like this.

![Deployed Cluster](docs/images/spark-cluster-aws.png)

### The Pipeline Platform PaaS

Although this readme is about the **Pipeline API** itself, in order to briefly show the full picture and understand how Pipeline is a core part and the engine behind, please find below a typical platform overview - again within the context of an out of the box Spark/Zeppelin spotguide.

![Pipeline PaaS](docs/images/spark-pipeline-aws.png)  

For the platform's end user a typical Pipeline interaction starts with a GitHub commit hook setup. Once the hook is configured (e.g. for branch merge, commit to master, etc.) the only task is to place the Pipeline Platform `descriptor` inside the GitHub repository. The `descriptor` contains information about the desired cluster type (size, kind, storage, etc), the autoscaling/SLA rules, custom monitoring rules (beside the default ones), authentication providers (Pipeline supports OAuth2, thus all providers with JWT token capabilities are supported).

### Installation

Pipeline API deploys microservice based applications using the REST API. Unlike the Pipeline Platform, the API has no explicit knowledge about `spotguides`, the CI/CD pipeline, or the application type. It merely provisions the cloud infrastructure (or reuses), deploys the containers through Helm and applies cluster reconciliation/SLA rules. In order to try the API follow the [installation guide](docs/install.md).

### Create and scale your cluster

Once Pipeline API is started, the easiest way to start, stop, delete or scale a cluster is through the following Postman example.

[![Run in Postman](https://run.pstmn.io/button.svg)](https://www.getpostman.com/collections/7a4c9291ff7b1afe5a5e)

For alternative ways to create a cluster please follow the [create cluster guide](docs/create.md).

### Quick howto

To do a step by step installation there is a detailed howto available [here](docs/pipeline-howto.md).

### `Spotguide` specification

In surfing a `spotguide` contains information about the wave, access, surrounding area, hazards and the best swell directions - basically all you need to know about the spot, in an organized and well maintained way.

>Spotguides are not part of the Pipeline project and this repository. They are nothing else but a collection of yaml files and custom CI/CD plugins written in Go.

For Pipeline a `spotguide` is a combination of a few yaml files that describe an application, specify the deployment environment, contain the SLA rules for resilience, autoscaling and failover scenarios, and  describe a runtime analyzer/monitor. `Spotguides` typically examine the application to determine the dependencies and have an understanding of how to build and deploy it. All the Kubernetes and cloud infrastructure related services are configured out of the box - e.g if a service needs ingress (based on the protocol and the application/deployment type) than the `spotguide` downloads, installs, starts and registers those service (e.g. Traefik, AWS ELB, etc).

When you push a code change to GitHub, the Pipeline platform automatically detects the appropriate `spotguide`, reads the descriptors and initiates the pipeline.

#### Big data

Pipeline PaaS allows enterprises to shift from a host-centric infrastructure to one that is container and application centric and take advantage of containers’ portability and flexibility. Today's big data frameworks require a scheduler (Apache YARN) and a distributed coordination framework (Apache Zookeeper) however better alternatives are already key building blocks of Kubernetes. Running big data workloads on the Pipeline PaaS removes all the requirements to use, install and maintain these systems and provide a cloud native way to run, schedule and scale the workload. The Kubernetes scheduler is aware of the application state and understands the infrastructure and cluster as well. A better density, utilization, broader range of workloads and varying latency  are all among the benefits.

#### Apache Spark

One of the default `spotguides` describes an Apache Spark deployment. For further information about the Apache Spark `spotguide` please follow this [guide](docs/spotguides.md).
A typical example of a Spark flow is this.

![Spark Flow](docs/images/spark-flow.png)

_Note: Spark on Kubernetes does not use YARN, all scheduling and resource management is natively and more efficiently done by the Kuberneres scheduler._

#### Apache Zeppelin

The Apache Zeppelin `spotguide` picks up a change in a Spark notebook and deploys and executes it on Kubernetes in cluster mode. For further information about the Apache Zeppelin `spotguide` please follow this [guide](docs/spotguides.md)
A typical example of a Zeppelin flow is this.

![Zeppelin Flow](docs/images/zeppelin-flow.png)

_Note: Zeppelin on Kubernetes for Spark notebooks does not use YARN, all scheduling and resource management is natively and more efficiently done by the Kuberneres scheduler._

#### Apache Kafka

The Apache Kafka `spotguide` has a good understanding of consumers and producers but more importantly it monitors, scales, rebalances and auto-heals the Kafka cluster. It autodetects broker failures, reassigns workloads and edits partition reassignment files.

_Note: Kafka on Kubernetes does not use Zookeper at all. For all quotas, controller election, cluster membership and configuration it is using **etcd**, a faster and more reliable `cloud-native` distributed system for coordination and metadata storage._

### Reporting bugs

In case you have problems please open an [issue](https://github.com/banzaicloud/pipeline/issues) on GitHub. Please note that _Pipeline is experimental, under development and does not have a stable release yet. If in doubt, don't go out._

### Contributing

Thanks you for your contribution and being part of our community. Please read [CONTRIBUTING.md](CONTRIBUTING.md) for details on the code of conduct, and the process for submitting pull requests. When you are opening a PR to Pipeline the first time we will require you to sign a standard CLA.

### License

Pipeline is open sourced under the Apache 2.0 license. See the [LICENSE](LICENSE.md) file for details.
