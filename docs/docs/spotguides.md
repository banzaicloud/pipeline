### Spotguide specification

#### Apache Spark

One of the default spotguide is the Apache Spark one. There are many examples of how to run Spark on Kuberneted/cloud, however they all have one common feature - either run on standalone or YARN mode. None of this is really beneficial, the Spark on YARN (deployed to cloud/Kubernetes) is especially a very bad example of having multiple resource managers without any knowledge about each other. Pipeline's spotguide beside that understand a Spark job it also understands the Apache Spark [internals](https://github.com/apache-spark-on-k8s/spark) - all the Spark components are deployed in containers, scheduled by the Kubernetes scheduler and cluster/vertical/horizontal (auto)scaled.

A typical example of a Spark flow looks like this.

![Spark Flow](docs/images/spark-flow.png)


#### Apache Zeppelin

The Apache Zeppelin spotguide picks up a change in a Spark notebook and deploys and executes it on Kubernetes/cloud in cluster mode. This is built on top of the Spark spotguide - the Spark Driver runs inside the Kubernetes cluster. Zeppelin uses spark-submit to start RemoteInterpreterServer which is able execute notebooks the notebooks on Spark. 

A typical example of a Zeppelin flow looks like this.

![Zeppelin Flow](docs/images/zeppelin-flow.png)

#### Apache Kafka 

The Apache Kafka `spotguide` has a good understanding of consumers and producers but more importantly it monitors, scales, rebalances and auto-heal the Kafka cluster. It autodetects broker failures, reassign workloads and edits partition reassignment files.

_Note: Kafka on Kubernetes does not use Zookeper at all. For all quotas, controller election, cluster membership and configuration is using **etcd**, a faster and more reliable `cloud-native` distributed system for coordination and metadata storage._
