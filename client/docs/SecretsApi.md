# \SecretsApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AddSecretTag**](SecretsApi.md#AddSecretTag) | **Put** /api/v1/orgs/{orgId}/secrets/{secretId}/tags/{tag} | Add a tag to a secret
[**AddSecrets**](SecretsApi.md#AddSecrets) | **Post** /api/v1/orgs/{orgId}/secrets | Add secrets
[**AllowedSecretsTypes**](SecretsApi.md#AllowedSecretsTypes) | **Get** /api/v1/allowed/secrets | List allowed secret types
[**AllowedSecretsTypesKeys**](SecretsApi.md#AllowedSecretsTypesKeys) | **Get** /api/v1/allowed/secrets/{type} | List required keys
[**DeleteSecretTag**](SecretsApi.md#DeleteSecretTag) | **Delete** /api/v1/orgs/{orgId}/secrets/{secretId}/tags/{tag} | Delete a tag from a secret
[**DeleteSecrets**](SecretsApi.md#DeleteSecrets) | **Delete** /api/v1/orgs/{orgId}/secrets/{secretId} | Delete secrets
[**GetSecret**](SecretsApi.md#GetSecret) | **Get** /api/v1/orgs/{orgId}/secrets/{secretId} | Get secret
[**GetSecretTags**](SecretsApi.md#GetSecretTags) | **Get** /api/v1/orgs/{orgId}/secrets/{secretId}/tags | Get the tags of a secret
[**GetSecrets**](SecretsApi.md#GetSecrets) | **Get** /api/v1/orgs/{orgId}/secrets | List secrets
[**UpdateSecrets**](SecretsApi.md#UpdateSecrets) | **Put** /api/v1/orgs/{orgId}/secrets/{secretId} | Update secrets
[**ValidateSecret**](SecretsApi.md#ValidateSecret) | **Get** /api/v1/orgs/{orgId}/secrets/{secretId}/validate | Validate secret


# **AddSecretTag**
> []string AddSecretTag(ctx, orgId, secretId, tag)
Add a tag to a secret

Add a tag to a secret

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **secretId** | **string**| Secret identification | 
  **tag** | **string**| Tag | 

### Return type

**[]string**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **AddSecrets**
> CreateSecretResponse AddSecrets(ctx, orgId, createSecretRequest, optional)
Add secrets

Adding secrets

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **createSecretRequest** | [**CreateSecretRequest**](CreateSecretRequest.md)|  | 
 **optional** | ***AddSecretsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a AddSecretsOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------


 **validate** | **optional.Bool**| validation is skipped or not | 

### Return type

[**CreateSecretResponse**](CreateSecretResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **AllowedSecretsTypes**
> map[string]AllowedSecretTypeResponse AllowedSecretsTypes(ctx, )
List allowed secret types

List allowed secret types and their required keys

### Required Parameters
This endpoint does not need any parameter.

### Return type

[**map[string]AllowedSecretTypeResponse**](AllowedSecretTypeResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **AllowedSecretsTypesKeys**
> AllowedSecretTypeResponse AllowedSecretsTypesKeys(ctx, type_)
List required keys

List required keys in the given secret type

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **type_** | **string**| Secret type | 

### Return type

[**AllowedSecretTypeResponse**](AllowedSecretTypeResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeleteSecretTag**
> DeleteSecretTag(ctx, orgId, secretId, tag)
Delete a tag from a secret

Delete a tag from a secret

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **secretId** | **string**| Secret identification | 
  **tag** | **string**| Tag | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeleteSecrets**
> DeleteSecrets(ctx, orgId, secretId)
Delete secrets

Deleting secrets

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **secretId** | **string**| Secret identification | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetSecret**
> SecretItem GetSecret(ctx, orgId, secretId)
Get secret

Get secret

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **secretId** | **string**| Secret identification | 

### Return type

[**SecretItem**](SecretItem.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetSecretTags**
> []string GetSecretTags(ctx, orgId, secretId)
Get the tags of a secret

Get the tags of a secret

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **secretId** | **string**| Secret identification | 

### Return type

**[]string**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetSecrets**
> []SecretItem GetSecrets(ctx, orgId, optional)
List secrets

Listing secrets

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
 **optional** | ***GetSecretsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a GetSecretsOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **type_** | **optional.String**| Secret&#39;s type to filter with | 
 **tags** | [**optional.Interface of []string**](string.md)| The selected tag to filter with | 
 **values** | **optional.Bool**| Marks if to present secret values or just the keys | 

### Return type

[**[]SecretItem**](SecretItem.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdateSecrets**
> CreateSecretResponse UpdateSecrets(ctx, orgId, secretId, createSecretRequest, optional)
Update secrets

Update secrets

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **secretId** | **string**| Secret identification | 
  **createSecretRequest** | [**CreateSecretRequest**](CreateSecretRequest.md)|  | 
 **optional** | ***UpdateSecretsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a UpdateSecretsOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



 **validate** | **optional.Bool**| validation is skipped or not | 

### Return type

[**CreateSecretResponse**](CreateSecretResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ValidateSecret**
> ValidateSecret(ctx, orgId, secretId)
Validate secret

Validate secret

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **secretId** | **string**| Secret identification | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

