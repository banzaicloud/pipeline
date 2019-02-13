# \SpotguidesApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**GetSpotguideDetail**](SpotguidesApi.md#GetSpotguideDetail) | **Get** /api/v1/orgs/{orgId}/spotguides/{name} | Get spotguide details
[**LaunchSpotguide**](SpotguidesApi.md#LaunchSpotguide) | **Post** /api/v1/orgs/{orgId}/spotguides | Launch spotguide
[**ListSpotguides**](SpotguidesApi.md#ListSpotguides) | **Get** /api/v1/orgs/{orgId}/spotguides | List spotguides
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

# **LaunchSpotguide**
> LaunchSpotguide(ctx, orgId, launchSpotguidesRequest)
Launch spotguide

Launch a spotguide

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **launchSpotguidesRequest** | [**LaunchSpotguidesRequest**](LaunchSpotguidesRequest.md)|  | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListSpotguides**
> []SpotguideDetailsResponse ListSpotguides(ctx, orgId)
List spotguides

List all available spotguides

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 

### Return type

[**[]SpotguideDetailsResponse**](SpotguideDetailsResponse.md)

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

