# \SpotguidesApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetSpotguideDetail**](SpotguidesApi.md#GetSpotguideDetail) | **Get** /api/v1/orgs/{orgId}/spotguides/{name} | Get spotguide details
[**UpdateSpotguides**](SpotguidesApi.md#UpdateSpotguides) | **Put** /api/v1/orgs/{orgId}/spotguides | Update spotguide repositories


# **GetSpotguideDetail**
> SpotguideDetailsResponse GetSpotguideDetail(ctx, orgId, name)
Get spotguide details

Get details about specific spotguide

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **name** | **string**| Spotguide name | 

### Return type

[**SpotguideDetailsResponse**](SpotguideDetailsResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **UpdateSpotguides**
> UpdateSpotguides(ctx, orgId)
Update spotguide repositories

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

