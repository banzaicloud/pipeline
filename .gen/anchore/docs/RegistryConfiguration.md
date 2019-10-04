# RegistryConfiguration

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**CreatedAt** | [**time.Time**](time.Time.md) |  | [optional] 
**LastUpated** | [**time.Time**](time.Time.md) |  | [optional] 
**RegistryUser** | **string** | Username portion of credential to use for this registry | [optional] 
**RegistryType** | **string** | Type of registry | [optional] 
**UserId** | **string** | Engine user that owns this registry entry | [optional] 
**Registry** | **string** | hostname:port string for accessing the registry, as would be used in a docker pull operation | [optional] 
**RegistryName** | **string** | human readable name associated with registry record | [optional] 
**RegistryVerify** | **bool** | Use TLS/SSL verification for the registry URL | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


