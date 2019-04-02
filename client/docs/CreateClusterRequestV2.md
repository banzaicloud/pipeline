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
**Kubernetes** | [**map[string]interface{}**](map[string]interface{}.md) |  | 
**Location** | **string** | Non-existent resources will be created in this location. Existing resources that must have the same location as the cluster will be validated against this. | [optional] 
**ResourceGroup** | **string** | Required resources will be created in this resource group. | 
**Network** | [**map[string]interface{}**](map[string]interface{}.md) |  | [optional] 
**Nodepools** | [**[]map[string]interface{}**](map[string]interface{}.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


