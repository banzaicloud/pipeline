# \ArkBackupsApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**CreateARKBackupOfACluster**](ArkBackupsApi.md#CreateARKBackupOfACluster) | **Post** /api/v1/orgs/{orgId}/clusters/{id}/backups | Create ARK backup
[**DeleteARKBackup**](ArkBackupsApi.md#DeleteARKBackup) | **Delete** /api/v1/orgs/{orgId}/clusters/{id}/backups/{backupId} | Delete ARK backup
[**DownloadARKBackupContents**](ArkBackupsApi.md#DownloadARKBackupContents) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/backups/{backupId}/download | Download ARK backup contents
[**GetARKBackup**](ArkBackupsApi.md#GetARKBackup) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/backups/{backupId} | Get ARK backup
[**GetARKBackupLogs**](ArkBackupsApi.md#GetARKBackupLogs) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/backups/{backupId}/logs | Get ARK backup logs
[**ListARKBackupsForOrganization**](ArkBackupsApi.md#ListARKBackupsForOrganization) | **Get** /api/v1/orgs/{orgId}/backups | List ARK backups of an Organization
[**ListARKBackupsOfACluster**](ArkBackupsApi.md#ListARKBackupsOfACluster) | **Get** /api/v1/orgs/{orgId}/clusters/{id}/backups | List ARK backups of a cluster
[**SyncARKBackupsOfACluster**](ArkBackupsApi.md#SyncARKBackupsOfACluster) | **Put** /api/v1/orgs/{orgId}/backups/{id}/sync | Sync ARK backups of a cluster
[**SyncOrgBackups**](ArkBackupsApi.md#SyncOrgBackups) | **Put** /api/v1/orgs/{orgId}/backups/sync | Sync ARK backups of an Organization



## CreateARKBackupOfACluster

> CreateBackupResponse CreateARKBackupOfACluster(ctx, orgId, id, createBackupRequest)
Create ARK backup

Create ARK backup of a cluster

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**id** | **int32**| Selected cluster identification (number) | 
**createBackupRequest** | [**CreateBackupRequest**](CreateBackupRequest.md)|  | 

### Return type

[**CreateBackupResponse**](CreateBackupResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DeleteARKBackup

> DeleteBackupResponse DeleteARKBackup(ctx, orgId, id, backupId)
Delete ARK backup

Delete ARK backup

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**id** | **int32**| Selected cluster identification (number) | 
**backupId** | **int32**| ID of the backup | 

### Return type

[**DeleteBackupResponse**](DeleteBackupResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## DownloadARKBackupContents

> *os.File DownloadARKBackupContents(ctx, orgId, id, backupId)
Download ARK backup contents

Download ARK backup contents

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**id** | **int32**| Selected cluster identification (number) | 
**backupId** | **int32**| ID of the backup | 

### Return type

[***os.File**](*os.File.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/x-gzip, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetARKBackup

> BackupResponse GetARKBackup(ctx, orgId, id, backupId)
Get ARK backup

Get ARK backup

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**id** | **int32**| Selected cluster identification (number) | 
**backupId** | **int32**| ID of the backup | 

### Return type

[**BackupResponse**](BackupResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## GetARKBackupLogs

> string GetARKBackupLogs(ctx, orgId, id, backupId)
Get ARK backup logs

Get ARK backup logs

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**id** | **int32**| Selected cluster identification (number) | 
**backupId** | **int32**| ID of the backup | 

### Return type

**string**

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: text/plain, application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListARKBackupsForOrganization

> []BackupResponse ListARKBackupsForOrganization(ctx, orgId)
List ARK backups of an Organization

List ARK backups of an Organization

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 

### Return type

[**[]BackupResponse**](BackupResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## ListARKBackupsOfACluster

> []BackupResponse ListARKBackupsOfACluster(ctx, orgId, id)
List ARK backups of a cluster

List ARK backups of a cluster

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**id** | **int32**| Selected cluster identification (number) | 

### Return type

[**[]BackupResponse**](BackupResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## SyncARKBackupsOfACluster

> SyncARKBackupsOfACluster(ctx, orgId, id)
Sync ARK backups of a cluster

Sync ARK backups of a cluster

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


## SyncOrgBackups

> SyncOrgBackups(ctx, orgId)
Sync ARK backups of an Organization

Sync ARK backups of an Organization

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

