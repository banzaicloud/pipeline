#### Deployments through REST

Behind the scene Pipeline is using Helm to deploy applications to Kubernetes. In order to interact with Helm (basically Tiller) you have to use the CLI or write gRPC code and talk directly with Tiller. For the Pipeline PaaS, and our end users we needed to build a REST based API to speak with the Tiller server. This implementation resides inside the [helm](https://github.com/banzaicloud/pipeline/tree/master/helm) package and it's exposed using a [Gin](https://github.com/gin-gonic/gin) server by [Pipeline](https://github.com/banzaicloud/pipeline) itself.

To flow of deploying a Helm chart to Kubernetes will look like this:

![](https://raw.githubusercontent.com/banzaicloud/pipeline/master/docs/images/tiller-rest-flow.png)

Now lets see how can you access and deploy a Kubernetes application using the REST endpoints with `curl`. The first thing you might need is a `kubeconfig` if you'd like to validate the deployment using the Helm CLI as well. Beside provisioning Kubernetes clusters in the cloud, [Pipeline](https://github.com/banzaicloud/pipeline) can connect to existing clusters, disregarding whether it's cloud or on-prem based. 

##### Get the Kubernetes config

In order to get the `kubeconfig` you can do the following REST call:

```
curl --request GET \
  --url 'http://{{url}}/api/v1/clusters/{{cluster_id}}/config' \
  --header 'Authorization: Bearer PIPELINE_TOKEN' \
  --header 'Content-Type: application/json'
```

##### Post a deployment

Now you can add a deployment with the following REST call:

```
curl --request POST \
  --url 'http://{{url}}/api/v1/clusters/{{cluster_id}}/deployments' \
  --header 'Authorization: Bearer PIPELINE_TOKEN' \
  --header 'Content-Type: application/json' \
  --data '{"name": "spark-shuffle"}'
```
##### Check deployment status

Once the deployment is posted you can check the status with this HEAD call:

```
curl --request HEAD \
  --url 'http://{{url}}/api/v1/clusters/{{cluster_id}}/deployments/{{deployment_name}}' \
  --header 'Authorization: Bearer PIPELINE_TOKEN' \
  --header 'Content-Type: application/json'
```
##### Upgrade a deployment

Deployments can be upgraded with the following PUT call:

```
curl --request PUT \
  --url 'http://{{url}}/api/v1/clusters/{{cluster_id}}/deployments/{{deployment_name}}' \
  --header 'Authorization: Bearer PIPELINE_TOKEN' \
  --header 'Content-Type: application/x-www-form-urlencoded'
```

##### Delete a deployment

Finally a deployment can be deleted as well:

```
curl --request DELETE \
  --url 'http://{{url}}/api/v1/clusters/{{cluster_id}}/deployments/{{deployment_name}}' \
  --header 'Authorization: Bearer PIPELINE_TOKEN' \
  --header 'Content-Type: application/x-www-form-urlencoded'
```

We use these REST API calls as well and have collected them in a **Postman** collection:

[![Run in Postman](https://run.pstmn.io/button.svg)](https://www.getpostman.com/collections/56684ef61ee236e8f30d)

Alternatively you can access the same collection online as well following this [link](https://documenter.getpostman.com/view/3197144/end2end-test-v020/7TDmb19#intro).

We have also created a Docker image that can be used to run the collection in a **containerized**  manner. Build the image from our GitHub [repository](https://github.com/banzaicloud/dockerized-newman) or pull it from the [Docker Hub](https://hub.docker.com/r/banzaicloud/dockerized-newman/).
