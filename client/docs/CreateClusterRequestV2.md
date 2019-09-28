# CreateClusterRequestV2

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** |  | 
**Features** | [**[]Feature**](Feature.md) |  | [optional] 
**SecretId** | **string** |  | [optional] 
**SecretName** | **string** |  | [optional] 
**SshSecretId** | **string** |  | [optional] 
**ScaleOptions** | [**ScaleOptions**](ScaleOptions.md) |  | [optional] 
**Type** | **string** |  | 
**Kubernetes** | [**CreatePkeClusterKubernetes**](CreatePKEClusterKubernetes.md) |  | 
**Location** | **string** | Non-existent resources will be created in this location. Existing resources that must have the same location as the cluster will be validated against this. | [optional] 
**ResourceGroup** | **string** | Required resources will be created in this resource group. | 
**Network** | [**PkeOnAzureClusterNetwork**](PKEOnAzureClusterNetwork.md) |  | [optional] 
**Nodepools** | [**[]PkeOnAzureNodePool**](PKEOnAzureNodePool.md) |  | [optional] 
**AccessPoints** | **[]string** | List of access points (i.e. load balancers, floating IPs) to be created for the cluster. Access points are implemented using cloud provider specific resources. | [optional] 
**ApiServerAccessPoints** | **[]string** | List of access point references for the API server; currently, public and private are the only valid values | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


