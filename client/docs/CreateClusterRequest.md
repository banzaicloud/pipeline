# CreateClusterRequest

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** |  | 
**Location** | **string** |  | 
**Cloud** | **string** |  | 
**TtlMinutes** | **int32** | The lifespan of the cluster expressed in minutes after which it is automatically deleted. Zero value means the cluster is never automatically deleted. | [optional] 
**SecretId** | **string** |  | [optional] 
**SecretIds** | **[]string** |  | [optional] 
**SecretName** | **string** |  | [optional] 
**PostHooks** | [**map[string]interface{}**](.md) |  | [optional] 
**ProfileName** | **string** |  | [optional] 
**ScaleOptions** | [**ScaleOptions**](ScaleOptions.md) |  | [optional] 
**Properties** | [**map[string]interface{}**](.md) |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


