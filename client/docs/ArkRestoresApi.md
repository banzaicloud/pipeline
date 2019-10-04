# \ArkRestoresApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateARKRestore**](ArkRestoresApi.md#CreateARKRestore) | **Post** /api/v1/orgs/{orgId}/clusters/{id}/restores | Create ARK restore
[**DeleteARKRestore**](ArkRestoresApi.md#DeleteARKRestore) | **Delete** /api/v1/orgs/{orgId}/clusters/{id}/restores/{restoreId} | Delete ARK restore
[**GetARKRestore**](ArkRestoresApi.md#GetARKRestore) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/restores/{restoreId} | Get ARK restore
[**GetARKRestoreResuts**](ArkRestoresApi.md#GetARKRestoreResuts) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/restores/{restoreId}/results | Get ARK restore results
[**ListARKRestores**](ArkRestoresApi.md#ListARKRestores) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/restores | List ARK restores
[**SyncARKRestoresOfACluster**](ArkRestoresApi.md#SyncARKRestoresOfACluster) | **Put** /api/v1/orgs/{orgId}/restores/{id}/sync | Sync ARK restores of a cluster



## CreateARKRestore

> CreateRestoreResponse CreateARKRestore(ctx, orgId, id, createRestoreRequest)
Create ARK restore

Create ARK restore

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**id** | **int32**| Selected cluster identification (number) | 
**createRestoreRequest** | [**CreateRestoreRequest**](CreateRestoreRequest.md)|  | 

### Return type

[**CreateRestoreResponse**](CreateRestoreResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteARKRestore

> DeleteRestoreResponse DeleteARKRestore(ctx, orgId, id, restoreId)
Delete ARK restore

Delete ARK restore

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**id** | **int32**| Selected cluster identification (number) | 
**restoreId** | **int32**| ID of the restore | 

### Return type

[**DeleteRestoreResponse**](DeleteRestoreResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetARKRestore

> RestoreResponse GetARKRestore(ctx, orgId, id, restoreId)
Get ARK restore

Get ARK restore

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**id** | **int32**| Selected cluster identification (number) | 
**restoreId** | **int32**| ID of the restore | 

### Return type

[**RestoreResponse**](RestoreResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetARKRestoreResuts

> RestoreResultsResponse GetARKRestoreResuts(ctx, orgId, id, restoreId)
Get ARK restore results

Get ARK restore results

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**id** | **int32**| Selected cluster identification (number) | 
**restoreId** | **int32**| ID of the restore | 

### Return type

[**RestoreResultsResponse**](RestoreResultsResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListARKRestores

> []RestoreResponse ListARKRestores(ctx, orgId, id)
List ARK restores

List ARK restores

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**id** | **int32**| Selected cluster identification (number) | 

### Return type

[**[]RestoreResponse**](RestoreResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SyncARKRestoresOfACluster

> SyncARKRestoresOfACluster(ctx, orgId, id)
Sync ARK restores of a cluster

Sync ARK restores of a cluster

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
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

