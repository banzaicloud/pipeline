# \ImagescanApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetImageVulnerabilities**](ImagescanApi.md#GetImageVulnerabilities) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/imagescan/{imageDigest}/vuln | Get vulnerabilities
[**GetScanResult**](ImagescanApi.md#GetScanResult) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/imagescan/{imageDigest} | Get Image scan results
[**ScanImages**](ImagescanApi.md#ScanImages) | **Post** /api/v1/orgs/{orgId}/clusters/{id}/imagescan | Scan Images used in cluster


# **GetImageVulnerabilities**
> VulnerabilityResponse GetImageVulnerabilities(ctx, orgId, id, imageDigest)
Get vulnerabilities

Get vulnerabilities

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **imageDigest** | **string**| Image digest | 

### Return type

[**VulnerabilityResponse**](VulnerabilityResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetScanResult**
> AnchoreImage GetScanResult(ctx, orgId, id, imageDigest)
Get Image scan results

Get Image scan result

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **imageDigest** | **string**| Image digest | 

### Return type

[**AnchoreImage**](AnchoreImage.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ScanImages**
> []AnchoreImage ScanImages(ctx, orgId, id, clusterImage)
Scan Images used in cluster

Scan Images used in cluster

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **clusterImage** | [**[]ClusterImage**](array.md)|  | 

### Return type

[**[]AnchoreImage**](AnchoreImage.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

