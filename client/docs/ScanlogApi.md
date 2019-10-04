# \ScanlogApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**ListScans**](ScanlogApi.md#ListScans) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/scanlog | List scans
[**ListScansByRelease**](ScanlogApi.md#ListScansByRelease) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/scanlog/{releaseName} | List scans by release



## ListScans

> []ScanLogItem ListScans(ctx, id, orgId)
List scans

List scans

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **int32**| Selected cluster identification (number) | 
**orgId** | **int32**| Organization identification | 

### Return type

[**[]ScanLogItem**](ScanLogItem.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListScansByRelease

> []ScanLogItem ListScansByRelease(ctx, id, orgId, releaseName)
List scans by release

List scans by release

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**id** | **int32**| Selected cluster identification (number) | 
**orgId** | **int32**| Organization identification | 
**releaseName** | **string**| release name identification | 

### Return type

[**[]ScanLogItem**](ScanLogItem.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

