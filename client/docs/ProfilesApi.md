# \ProfilesApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**AddProfiles**](ProfilesApi.md#AddProfiles) | **Post** /api/v1/orgs/{orgId}/profiles/cluster | Add cluster profiles
[**DeleteProfiles**](ProfilesApi.md#DeleteProfiles) | **Delete** /api/v1/orgs/{orgId}/profiles/cluster/{type}/{name} | Delete cluster profiles
[**ListProfiles**](ProfilesApi.md#ListProfiles) | **Get** /api/v1/orgs/{orgId}/profiles/cluster/{type} | List cluster profiles
[**UpdateProfiles**](ProfilesApi.md#UpdateProfiles) | **Put** /api/v1/orgs/{orgId}/profiles/cluster | Update cluster profiles


# **AddProfiles**
> AddProfiles(ctx, orgId, addClusterProfileRequest)
Add cluster profiles

Add cluster profile

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **addClusterProfileRequest** | [**AddClusterProfileRequest**](AddClusterProfileRequest.md)|  | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeleteProfiles**
> DeleteProfiles(ctx, orgId, type_, name)
Delete cluster profiles

Delete cluster profiles by cloud type and name

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **type_** | **string**| Cloud type | 
  **name** | **string**| Cluster profile name | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListProfiles**
> ProfileListResponse ListProfiles(ctx, orgId, type_)
List cluster profiles

Listing cluster profiles by cloud type

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **type_** | **string**| Cloud type | 

### Return type

[**ProfileListResponse**](ProfileListResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdateProfiles**
> UpdateProfiles(ctx, orgId, addClusterProfileRequest)
Update cluster profiles

Update an existing cluster profile

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **addClusterProfileRequest** | [**AddClusterProfileRequest**](AddClusterProfileRequest.md)|  | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

