# CreatePkeOnAzureClusterRequestAllOf

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Location** | **string** | Non-existent resources will be created in this location. Existing resources that must have the same location as the cluster will be validated against this. | [optional] 
**ResourceGroup** | **string** | Required resources will be created in this resource group. | 
**Network** | [**PkeOnAzureClusterNetwork**](PKEOnAzureClusterNetwork.md) |  | [optional] 
**Nodepools** | [**[]PkeOnAzureNodePool**](PKEOnAzureNodePool.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


