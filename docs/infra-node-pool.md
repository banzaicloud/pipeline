### Setup reserved node pool for Pipeline Infra components 

You can specify a reserved node pool for Infra deployments in Pipeline config for example: headNodePoolName="head". 
If you set this property Pipeline will place a taint on all nodes in this node pool and Tiller will be deployed with
a node selector and toleration matching nodes from this node pool. 
Make sure all Infra deployments are setup with the following node-affinity and toleration:


```
        spec:
        ...
          affinity:
            nodeAffinity:
              requiredDuringSchedulingIgnoredDuringExecution:
                nodeSelectorTerms:
                  - matchExpressions:
                    - key: nodepool.banzaicloud.io/name
                      operator: In
                      values:
                      - headNodePoolName
                      
          tolerations:
            - key: nodepool.banzaicloud.io/name
              operator: Equal
              value: "headNodePoolName"
```

> Pipeline doesn't create this node pool automatically for you, you need to create a node pool with your selected name set in `infra.headNodePoolName` with `Create cluster` request. 
