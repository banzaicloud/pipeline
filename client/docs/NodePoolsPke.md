# NodePoolsPke

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** |  | 
**Roles** | **[]string** |  | 
**Labels** | **map[string]string** | user provided custom node labels to be placed onto the nodes of the node pool | [optional] 
**Autoscaling** | **bool** | Enables/disables autoscaling of this node pool through Kubernetes cluster autoscaler. | 
**Provider** | **string** |  | 
**ProviderConfig** | [**map[string]interface{}**](.md) |  | 
**Hosts** | [**[]PkeHosts**](PKEHosts.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


