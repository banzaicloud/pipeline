# \StorageApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateObjectStoreBucket**](StorageApi.md#CreateObjectStoreBucket) | **Post** /api/v1/orgs/{orgId}/buckets | Creates a new object store bucket with the given params
[**DeleteObjectStoreBucket**](StorageApi.md#DeleteObjectStoreBucket) | **Delete** /api/v1/orgs/{orgId}/buckets/{name} | Deletes the object store bucket with the given name
[**GetObjectStoreBucketStatus**](StorageApi.md#GetObjectStoreBucketStatus) | **Head** /api/v1/orgs/{orgId}/buckets/{name} | Get object store bucket status
[**ListObjectStoreBuckets**](StorageApi.md#ListObjectStoreBuckets) | **Get** /api/v1/orgs/{orgId}/buckets | List object storage buckets


# **CreateObjectStoreBucket**
> CreateObjectStoreBucketResponse CreateObjectStoreBucket(ctx, orgId, createObjectStoreBucketRequest)
Creates a new object store bucket with the given params

Creates a new object store bucket on the Cloud provider referenced by the given secret. The credentials for creating the bucket is taken from the provided secret.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **createObjectStoreBucketRequest** | [**CreateObjectStoreBucketRequest**](CreateObjectStoreBucketRequest.md)|  | 

### Return type

[**CreateObjectStoreBucketResponse**](CreateObjectStoreBucketResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: application/json
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **DeleteObjectStoreBucket**
> DeleteObjectStoreBucket(ctx, orgId, name, secretId, cloudType, optional)
Deletes the object store bucket with the given name

Deletes the object store bucket identified by the given name. The credentials for deleting the bucket is taken from the provided secret.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **name** | **string**| Bucket identification | 
  **secretId** | **string**| Secret identification | 
  **cloudType** | **string**| Identifies the cloud provider | 
 **optional** | ***DeleteObjectStoreBucketOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a DeleteObjectStoreBucketOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------




 **resourceGroup** | **optional.String**| Azure resource group the storage account that holds the bucket (storage container) to be deleted | 
 **storageAccount** | **optional.String**| Azure storage account to delete the bucket (storage container) from | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **GetObjectStoreBucketStatus**
> GetObjectStoreBucketStatus(ctx, orgId, name, secretId, cloudType, optional)
Get object store bucket status

Retrieves the status of the object store bucket given its name

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **name** | **int32**| Bucket identification | 
  **secretId** | **string**| Secret identification | 
  **cloudType** | **string**| Identifies the cloud provider | 
 **optional** | ***GetObjectStoreBucketStatusOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a GetObjectStoreBucketStatusOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------




 **resourceGroup** | **optional.String**| Azure resource group to lookup the bucket(storage container) under. Required only on Azure cloud provider. | 
 **storageAccount** | **optional.String**| Azure storage account to lookup the bucket(storage container) under. Required only on Azure cloud provider. | 
 **location** | **optional.String**| The region to lookup the bucket under. Required on Amazon, Oracle and Alibaba cloud providers. | 

### Return type

 (empty response body)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: Not defined

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

# **ListObjectStoreBuckets**
> ListStorageBucketsResponse ListObjectStoreBuckets(ctx, orgId, secretId, cloudType, optional)
List object storage buckets

List object store buckets accessible by the credentials referenced by the given secret.

### Required Parameters

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
 **ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
  **orgId** | **int32**| Organization identification | 
  **secretId** | **string**| Secret identification | 
  **cloudType** | **string**| Identifies the cloud provider | 
 **optional** | ***ListObjectStoreBucketsOpts** | optional parameters | nil if no parameters

### Optional Parameters
Optional parameters are passed through a pointer to a ListObjectStoreBucketsOpts struct

Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



 **location** | **optional.String**| Identifies the cloud region. Required by Amazon only. | 

### Return type

[**ListStorageBucketsResponse**](ListStorageBucketsResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

 - **Content-Type**: Not defined
 - **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to Model list]](../README.md#documentation-for-models) [[Back to README]](../README.md)

