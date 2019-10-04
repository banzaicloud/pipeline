# RegistryConfigurationRequest

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**RegistryUser** | **string** | Username portion of credential to use for this registry | [optional] 
**RegistryPass** | **string** | Password portion of credential to use for this registry | [optional] 
**RegistryType** | **string** | Type of registry | [optional] 
**Registry** | **string** | hostname:port string for accessing the registry, as would be used in a docker pull operation. May include some or all of a repository and wildcards (e.g. docker.io/library/_* or gcr.io/myproject/myrepository) | [optional] 
**RegistryName** | **string** | human readable name associated with registry record | [optional] 
**RegistryVerify** | **bool** | Use TLS/SSL verification for the registry URL | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


