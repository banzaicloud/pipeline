# \ArkBucketsApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateBackupBucket**](ArkBucketsApi.md#CreateBackupBucket) | **Post** /api/v1/orgs/{orgId}/backupbuckets | Create backup bucket
[**DeleteBackupBucket**](ArkBucketsApi.md#DeleteBackupBucket) | **Delete** /api/v1/orgs/{orgId}/backupbuckets/{bucketId} | Delete backup bucket by ID
[**GetBackupBucket**](ArkBucketsApi.md#GetBackupBucket) | **Get** /api/v1/orgs/{orgId}/backupbuckets/{bucketId} | Get backup bucket by ID
[**ListBackupBuckets**](ArkBucketsApi.md#ListBackupBuckets) | **Get** /api/v1/orgs/{orgId}/backupbuckets | List backup buckets
[**SyncBackupBucket**](ArkBucketsApi.md#SyncBackupBucket) | **Put** /api/v1/orgs/{orgId}/backupbuckets/sync | Sync backup buckets



## CreateBackupBucket

> BackupBucketResponse CreateBackupBucket(ctx, orgId, createBackupBucketRequest)
Create backup bucket

Create backup bucket

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**createBackupBucketRequest** | [**CreateBackupBucketRequest**](CreateBackupBucketRequest.md)|  | 

### Return type

[**BackupBucketResponse**](BackupBucketResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteBackupBucket

> DeleteBackupBucketResponse DeleteBackupBucket(ctx, orgId, bucketId)
Delete backup bucket by ID

Delete backup bucket by ID

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**bucketId** | **int32**| ID of the bucket | 

### Return type

[**DeleteBackupBucketResponse**](DeleteBackupBucketResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetBackupBucket

> DeployedBackupBucketResponse GetBackupBucket(ctx, orgId, bucketId)
Get backup bucket by ID

Get backup bucket by ID

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**bucketId** | **int32**| ID of the bucket | 

### Return type

[**DeployedBackupBucketResponse**](DeployedBackupBucketResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListBackupBuckets

> []BackupBucketResponse ListBackupBuckets(ctx, orgId)
List backup buckets

List backup buckets

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 

### Return type

[**[]BackupBucketResponse**](BackupBucketResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SyncBackupBucket

> SyncBackupBucket(ctx, orgId)
Sync backup buckets

Sync backup buckets

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
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

