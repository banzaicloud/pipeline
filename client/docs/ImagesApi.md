# \ImagesApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ListDeploymentsByImage**](ImagesApi.md#ListDeploymentsByImage) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/images/{imageDigest}/deployments | List Deployments by Image
[**ListImages**](ImagesApi.md#ListImages) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/images | List Images used in cluster


# **ListDeploymentsByImage**
> []ListDeploymentsResponseItem ListDeploymentsByImage(ctx, orgId, id, imageDigest)
List Deployments by Image

List Deployments by Image

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **imageDigest** | **string**| Image digest | 

### Return type

[**[]ListDeploymentsResponseItem**](ListDeploymentsResponseItem.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListImages**
> []ClusterImage ListImages(ctx, orgId, id)
List Images used in cluster

List Images used in cluster

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

[**[]ClusterImage**](ClusterImage.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

