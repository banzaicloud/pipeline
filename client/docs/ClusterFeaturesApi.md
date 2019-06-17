# \ClusterFeaturesApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ActivateClusterFeature**](ClusterFeaturesApi.md#ActivateClusterFeature) | **Post** /api/v1/orgs/{orgId}/clusters/{id}/features/{featureName} | Activate a cluster feature
[**ClusterFeatureDetails**](ClusterFeaturesApi.md#ClusterFeatureDetails) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/features/{featureName} | Get details of a cluster feature
[**DeactivateClusterFeature**](ClusterFeaturesApi.md#DeactivateClusterFeature) | **Delete** /api/v1/orgs/{orgId}/clusters/{id}/features/{featureName} | Deactivate a cluster feature
[**ListClusterFeatures**](ClusterFeaturesApi.md#ListClusterFeatures) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/features | List enabled features of a cluster
[**UpdateClusterFeature**](ClusterFeaturesApi.md#UpdateClusterFeature) | **Put** /api/v1/orgs/{orgId}/clusters/{id}/features/{featureName} | Update a cluster feature


# **ActivateClusterFeature**
> ActivateClusterFeature(ctx, orgId, id, featureName, activateClusterFeatureRequest)
Activate a cluster feature

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization ID | 
  **id** | **int32**| Cluster ID | 
  **featureName** | **string**| Feature name | 
  **activateClusterFeatureRequest** | [**ActivateClusterFeatureRequest**](ActivateClusterFeatureRequest.md)|  | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ClusterFeatureDetails**
> ClusterFeatureDetails ClusterFeatureDetails(ctx, orgId, id, featureName)
Get details of a cluster feature

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization ID | 
  **id** | **int32**| Cluster ID | 
  **featureName** | **string**| Feature name | 

### Return type

[**ClusterFeatureDetails**](ClusterFeatureDetails.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeactivateClusterFeature**
> DeactivateClusterFeature(ctx, orgId, id, featureName)
Deactivate a cluster feature

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization ID | 
  **id** | **int32**| Cluster ID | 
  **featureName** | **string**| Feature name | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListClusterFeatures**
> map[string]ClusterFeatureDetails ListClusterFeatures(ctx, orgId, id)
List enabled features of a cluster

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization ID | 
  **id** | **int32**| Cluster ID | 

### Return type

[**map[string]ClusterFeatureDetails**](ClusterFeatureDetails.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdateClusterFeature**
> UpdateClusterFeature(ctx, orgId, id, featureName, updateClusterFeatureRequest)
Update a cluster feature

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization ID | 
  **id** | **int32**| Cluster ID | 
  **featureName** | **string**| Feature name | 
  **updateClusterFeatureRequest** | [**UpdateClusterFeatureRequest**](UpdateClusterFeatureRequest.md)|  | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

