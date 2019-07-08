# UpdateNodePoolsPke

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**InstanceType** | **string** | Instance type for the nodes in the node pool. This field is ignored when existing node pool is updated as instance type can not be changed. | [optional] 
**SpotPrice** | **string** | The spot price for a node in the node pool. Provide \&quot;\&quot; or \&quot;0\&quot; for on-demand nodes. | [optional] 
**Autoscaling** | **bool** | Whether to enable Kubernetes cluster autoscaler for this node pool. | [optional] 
**MinCount** | **int32** | If cluster autoscaler is enabled for this node pool it sets the minimum node count the cluster autoscaler can downscale the node pool to. | [optional] 
**MaxCount** | **int32** | If cluster autoscaler is enabled for this node pool it sets the maximum node count the cluster autoscaler can upscale the node pool to. | [optional] 
**Count** | **int32** | If cluster autoscaler is not enabled this specifies the desired ndoe count in the node pool. If cluster autoscaler is enabled this specifies the initial node count in the ndoe pool. | [optional] 
**Subnets** | **[]string** | The subnet to create the node pool into. If this field is omitted than the subnet from the cluster level network configuration is used. | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


