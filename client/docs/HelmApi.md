# \HelmApi

All URIs are relative to *http://localhost:9090*

Method | HTTP request | Description
------------- | ------------- | -------------
[**HelmChartDetails**](HelmApi.md#HelmChartDetails) | **Get** /api/v1/orgs/{orgId}/helm/chart/{repoName}/{chartName} | Chart details
[**HelmChartList**](HelmApi.md#HelmChartList) | **Get** /api/v1/orgs/{orgId}/helm/charts/ | Chart List
[**HelmListRepos**](HelmApi.md#HelmListRepos) | **Get** /api/v1/orgs/{orgId}/helm/repos | List repositories
[**HelmReposAdd**](HelmApi.md#HelmReposAdd) | **Post** /api/v1/orgs/{orgId}/helm/repos | Add Repo
[**HelmReposDelete**](HelmApi.md#HelmReposDelete) | **Delete** /api/v1/orgs/{orgId}/helm/repos/{repoName} | Delete Repo
[**HelmReposModify**](HelmApi.md#HelmReposModify) | **Put** /api/v1/orgs/{orgId}/helm/repos/{repoName} | Modify Repo
[**HelmReposUpdate**](HelmApi.md#HelmReposUpdate) | **Put** /api/v1/orgs/{orgId}/helm/repos/{repoName}/update | Update Repo



## HelmChartDetails

> HelmChartDetailsResponse HelmChartDetails(ctx, orgId, repoName, chartName, optional)
Chart details

Get helm chart details

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**repoName** | **string**| Chart repository name | 
**chartName** | **string**| Chart Name | 
 **optional** | ***HelmChartDetailsOpts** | optional parameters | nil if no parameters

### Optional Parameters

Optional parameters are passed through a pointer to a HelmChartDetailsOpts struct


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------



 **version** | **optional.String**| Chart version | 

### Return type

[**HelmChartDetailsResponse**](HelmChartDetailsResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## HelmChartList

> []map[string]interface{} HelmChartList(ctx, orgId, optional)
Chart List

Get available Helm chart's list

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
 **optional** | ***HelmChartListOpts** | optional parameters | nil if no parameters

### Optional Parameters

Optional parameters are passed through a pointer to a HelmChartListOpts struct


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------

 **name** | **optional.String**| Chart Name | 
 **repo** | **optional.String**| Repo Name | 
 **version** | **optional.String**| Chart version (latest) | 

### Return type

[**[]map[string]interface{}**](map[string]interface{}.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## HelmListRepos

> []HelmRepoListItem HelmListRepos(ctx, orgId)
List repositories

Listing Helm repositories in the cluster

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 

### Return type

[**[]HelmRepoListItem**](HelmRepoListItem.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## HelmReposAdd

> HelmRepoListItem HelmReposAdd(ctx, orgId, helmReposAddRequest)
Add Repo

Add new Helm repository

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**helmReposAddRequest** | [**HelmReposAddRequest**](HelmReposAddRequest.md)|  | 

### Return type

[**HelmRepoListItem**](HelmRepoListItem.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## HelmReposDelete

> HelmReposDeleteResponse HelmReposDelete(ctx, orgId, repoName)
Delete Repo

Delete  Helm repository

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**repoName** | **string**| Helm repo name | 

### Return type

[**HelmReposDeleteResponse**](HelmReposDeleteResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## HelmReposModify

> HelmReposUpdateResponse HelmReposModify(ctx, orgId, repoName, helmReposModifyRequest)
Modify Repo

Modify Helm repository

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**repoName** | **string**| Helm repo name | 
**helmReposModifyRequest** | [**HelmReposModifyRequest**](HelmReposModifyRequest.md)|  | 

### Return type

[**HelmReposUpdateResponse**](HelmReposUpdateResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: application/json
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)


## HelmReposUpdate

> HelmReposUpdateResponse HelmReposUpdate(ctx, orgId, repoName)
Update Repo

Modify Helm repository

### Required Parameters


Name | Type | Description  | Notes
------------- | ------------- | ------------- | -------------
**ctx** | **context.Context** | context for authentication, logging, cancellation, deadlines, tracing, etc.
**orgId** | **int32**| Organization identification | 
**repoName** | **string**| Helm repo name | 

### Return type

[**HelmReposUpdateResponse**](HelmReposUpdateResponse.md)

### Authorization

[bearerAuth](../README.md#bearerAuth)

### HTTP request headers

- **Content-Type**: Not defined
- **Accept**: application/json

[[Back to top]](#) [[Back to API list]](../README.md#documentation-for-api-endpoints)
[[Back to Model list]](../README.md#documentation-for-models)
[[Back to README]](../README.md)

