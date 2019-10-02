# \RegistriesApi

All URIs are relative to *http://localhost*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateRegistry**](RegistriesApi.md#CreateRegistry) | **Post** /registries | Add a new registry
[**DeleteRegistry**](RegistriesApi.md#DeleteRegistry) | **Delete** /registries/{registry} | Delete a registry configuration
[**GetRegistry**](RegistriesApi.md#GetRegistry) | **Get** /registries/{registry} | Get a specific registry configuration
[**ListRegistries**](RegistriesApi.md#ListRegistries) | **Get** /registries | List configured registries
[**UpdateRegistry**](RegistriesApi.md#UpdateRegistry) | **Put** /registries/{registry} | Update/replace a registry configuration


# **CreateRegistry**
> []RegistryConfiguration CreateRegistry(ctx, registrydata, optional)
Add a new registry

Adds a new registry to the system

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **registrydata** | [**RegistryConfigurationRequest**](RegistryConfigurationRequest.md)|  | 
 **optional** | ***CreateRegistryOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a CreateRegistryOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **validate** | **optional.Bool**| flag to determine whether or not to validate registry/credential at registry add time | 
 **xAnchoreAccount** | **optional.String**| An account name to change the resource scope of the request to that account, if permissions allow (admin only) | 

### Return type

[**[]RegistryConfiguration**](RegistryConfiguration.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeleteRegistry**
> DeleteRegistry(ctx, registry, optional)
Delete a registry configuration

Delete a registry configuration record from the system. Does not remove any images.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **registry** | **string**|  | 
 **optional** | ***DeleteRegistryOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a DeleteRegistryOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **xAnchoreAccount** | **optional.String**| An account name to change the resource scope of the request to that account, if permissions allow (admin only) | 

### Return type

 (empty response body)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetRegistry**
> RegistryConfiguration GetRegistry(ctx, registry, optional)
Get a specific registry configuration

Get information on a specific registry

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **registry** | **string**|  | 
 **optional** | ***GetRegistryOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a GetRegistryOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **xAnchoreAccount** | **optional.String**| An account name to change the resource scope of the request to that account, if permissions allow (admin only) | 

### Return type

[**RegistryConfiguration**](RegistryConfiguration.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListRegistries**
> []RegistryConfiguration ListRegistries(ctx, optional)
List configured registries

List all configured registries the system can/will watch

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
 **optional** | ***ListRegistriesOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ListRegistriesOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **xAnchoreAccount** | **optional.String**| An account name to change the resource scope of the request to that account, if permissions allow (admin only) | 

### Return type

[**[]RegistryConfiguration**](RegistryConfiguration.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdateRegistry**
> []RegistryConfiguration UpdateRegistry(ctx, registry, registrydata, optional)
Update/replace a registry configuration

Replaces an existing registry record with the given record

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **registry** | **string**|  | 
  **registrydata** | [**RegistryConfigurationRequest**](RegistryConfigurationRequest.md)|  | 
 **optional** | ***UpdateRegistryOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UpdateRegistryOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **validate** | **optional.Bool**| flag to determine whether or not to validate registry/credential at registry update time | 
 **xAnchoreAccount** | **optional.String**| An account name to change the resource scope of the request to that account, if permissions allow (admin only) | 

### Return type

[**[]RegistryConfiguration**](RegistryConfiguration.md)

### Authorization

[basicAuth](../README.md#basicAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

