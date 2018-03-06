### Create and scale your cluster 

Once Pipeline API is started you can create a cluster in the following ways: 

- Using the Postman examples
[![Run in Postman](https://run.pstmn.io/button.svg)](https://www.getpostman.com/collections/7a4c9291ff7b1afe5a5e)

- Using `make create-cluster` and `make delete-cluster`. If you have AWS CLI installed you can use `make ec2-list-instances` in order to see the list of clusters

- Setup your Pipeline GitHub OAuth application according to: [this guilde](./github-app.md)

- Acquire a Pipeline access token on the GUI via logging in at `http://localhost:9090/auth/github/login` and then after visit `http://localhost:9090/api/v1/token`. Save it into the `PIPELINE_TOKEN` environment variable.

- Use CURL `curl http://localhost:9090/api/v1/clusters -X POST -H "Authorization: Bearer $PIPELINE_TOKEN" -d name=test-$(USER) -d location=eu-west-1 -d nodeInstanceType=m4.xlarge -d nodeInstanceSpotPrice=0.2 -d nodeMin=1 -d nodeMax=3 -d image=ami-6d48500b`
    
_Note: AWS spot prices are supported_ 

Always use the **delete** feature of Pipeline. In case for some reason you can't delete the cluster through Pipeline, please remove the resources from the Cloud manually by starting with the **AutoScalingGroup** - otherwise AWS will relaunch the instances.
Pipeline creates, manages and deletes cloud resources for you. In order to avoid accidental exits and leave unremoved cloud resources, `CTRL-C` like exists are not supported. Should you want to exit Pipeline you should get the process ID and kill it manually.

### Scaling your cluster

Substitute <clusterid> with the ID of your cluster:
    
`curl -i -X PUT -H "Authorization: Bearer $PIPELINE_TOKEN" http://localhost:9090/api/v1/clusters/<clusterid> -H "Accept: application/json" -H "Content-Type: application/json" -d '{"node":{"minCount":6,"maxCount":12}}'`

### Logs

Currently Pipeline runs at the highest log level - in case of any problems please collect the logs and open an issue.
