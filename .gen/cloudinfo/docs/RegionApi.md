# \RegionApi

All URIs are relative to *http://localhost/api/v1*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetRegion**](RegionApi.md#GetRegion) | **Get** /providers/{provider}/services/{service}/regions/{region} | 



## GetRegion

> GetRegionResp GetRegion(ctx, provider, service, region)



Provides the detailed info of a specific region of a cloud provider

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**provider** | **string**|  | 
**service** | **string**|  | 
**region** | **string**|  | 

### Return type

[**GetRegionResp**](GetRegionResp.md)

### Authorization

No authorization required

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

