# \ArkApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CheckARKStatus**](ArkApi.md#CheckARKStatus) | **Head** /api/v1/orgs/{orgId}/clusters/{id}/backupservice/status | Check ARK service status
[**DisableARK**](ArkApi.md#DisableARK) | **Post** /api/v1/orgs/{orgId}/clusters/{id}/backupservice/disable | Disable ARK service
[**EnableARK**](ArkApi.md#EnableARK) | **Post** /api/v1/orgs/{orgId}/clusters/{id}/backupservice/enable | Enable ARK service


# **CheckARKStatus**
> CheckARKStatus(ctx, orgId, id)
Check ARK service status

Check ARK service status

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DisableARK**
> DisableArkResponse DisableARK(ctx, orgId, id)
Disable ARK service

Disable ARK service

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 

### Return type

[**DisableArkResponse**](DisableARKResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **EnableARK**
> EnableArkResponse EnableARK(ctx, orgId, id, enableArkRequest)
Enable ARK service

Enable ARK service

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **id** | **int32**| Selected cluster identification (number) | 
  **enableArkRequest** | [**EnableArkRequest**](EnableArkRequest.md)|  | 

### Return type

[**EnableArkResponse**](EnableARKResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

