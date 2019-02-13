# CreateGkePropertiesGke

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ProjectId** | **string** |  | [optional] 
**Master** | [**CreateGkePropertiesGkeMaster**](CreateGKEProperties_gke_master.md) |  | [optional] 
**NodeVersion** | **string** |  | [optional] 
**Vpc** | **string** | Name of the GCP Network (VPC) to deploy the cluster to. If omitted than the \&quot;default\&quot; VPC is used. | [optional] 
**Subnet** | **string** | Name of the GCP Subnet to deploy the cluster to. If \&quot;default\&quot; VPC is used this field can be omitted. The subnet must be in the same region as the location of the cluster. | [optional] 
**NodePools** | [**map[string]NodePoolsGoogle**](NodePoolsGoogle.md) |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


